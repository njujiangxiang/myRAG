package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"myrag/internal/config"
	"myrag/internal/database"
	"myrag/internal/embedding"
	"myrag/internal/handler"
	"myrag/internal/minio"
	"myrag/internal/models"
	"myrag/internal/nats"
	"myrag/internal/qdrant"
	"myrag/internal/rag"
	"myrag/internal/worker"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	fmt.Println("myRAG - Knowledge Base with AI Chat")
	fmt.Println("====================================")

	// 初始化日志器
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load configuration", zap.Error(err))
	}
	logger.Info("configuration loaded", zap.String("env", cfg.Server.Env))

	// 初始化数据库
	db, err := database.New(cfg.Database.URL, logger)
	if err != nil {
		logger.Fatal("failed to initialize database", zap.Error(err))
	}
	defer db.Close()

	// 初始化 Qdrant 客户端
	qdrantClient, err := qdrant.New(cfg.Qdrant.URL, cfg.Qdrant.APIKey, "documents", logger)
	if err != nil {
		logger.Fatal("failed to initialize Qdrant", zap.Error(err))
	}
	defer qdrantClient.Close()

	// 初始化 MinIO 客户端
	minioClient, err := minio.New(
		cfg.MinIO.Endpoint,
		cfg.MinIO.AccessKey,
		cfg.MinIO.SecretKey,
		cfg.MinIO.Bucket,
		cfg.MinIO.UseSSL,
		logger,
	)
	if err != nil {
		logger.Fatal("failed to initialize MinIO", zap.Error(err))
	}

	// 初始化 NATS 客户端
	natsClient, err := nats.New(cfg.NATS.URL, logger)
	if err != nil {
		logger.Fatal("failed to initialize NATS", zap.Error(err))
	}
	defer natsClient.Close()

	// 确保 NATS 流存在
	if err := natsClient.EnsureStream(context.Background(), "documents"); err != nil {
		logger.Warn("failed to ensure NATS stream", zap.Error(err))
	}

	// 初始化仓库
	userRepo := models.NewUserRepository(db.DB)
	kbRepo := models.NewKnowledgeBaseRepository(db.DB)
	docRepo := models.NewDocumentRepository(db.DB)
	sessionRepo := models.NewChatSessionRepository(db.DB)
	messageRepo := models.NewMessageRepository(db.DB)

	// 初始化嵌入客户端
	embeddingConfig := embedding.DefaultConfig()
	embeddingClient := embedding.New(embeddingConfig)
	logger.Info("embedding client initialized", zap.String("model", embeddingClient.GetModel()))

	// 初始化 RAG 工厂
	bm25IndexPath := "./data/rag-index/bm25"
	ragFactory := rag.NewFactory(rag.FactoryConfig{
		QdrantClient:    qdrantClient,
		EmbeddingClient: embeddingClient,
		LLMAPIKey:       cfg.LLM.APIKey,
		LLMModel:        cfg.LLM.Model,
		LLMProvider:     cfg.LLM.Provider,
		BM25IndexPath:   bm25IndexPath,
		Rerank: &rag.RerankConfig{
			Enabled:    cfg.Rerank.Enabled,
			BaseURL:    cfg.Rerank.BaseURL,
			Model:      cfg.Rerank.Model,
			TopK:       cfg.Rerank.TopK,
			Candidates: cfg.Rerank.Candidates,
		},
	})
	logger.Info("RAG factory initialized", zap.Any("strategies", ragFactory.ListStrategies()))

	// 初始化 handlers
	authHandler := handler.NewAuthHandler(userRepo, cfg.JWT.Secret)
	kbHandler := handler.NewKnowledgeBaseHandler(kbRepo)
	docHandler := handler.NewDocumentHandler(docRepo, kbRepo, minioClient, natsClient, qdrantClient)
	chatHandler := handler.NewChatHandler(sessionRepo, messageRepo, kbRepo, qdrantClient, embeddingClient, ragFactory, cfg.LLM.APIKey, cfg.LLM.Model, cfg.LLM.Provider)
	globalChatHandler := handler.NewGlobalChatHandler(sessionRepo, messageRepo, kbRepo, qdrantClient, embeddingClient, cfg.LLM.APIKey, cfg.LLM.Model, cfg.LLM.Provider)
	searchHandler := handler.NewSearchHandler(docRepo, qdrantClient, embeddingClient)

	// 初始化并启动文档 worker
	docWorker, err := worker.NewWorker(docRepo, minioClient, qdrantClient, embeddingClient, natsClient.JetStream)
	if err != nil {
		logger.Fatal("failed to initialize document worker", zap.Error(err))
	}
	go docWorker.Start(context.Background())

	// 设置 Gin 路由
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// 健康检查端点
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().UTC(),
		})
	})

	// API v1 路由
	v1 := router.Group("/api/v1")
	{
		// 认证路由（无需授权）
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/logout", authHandler.Logout)
			auth.GET("/me", handler.AuthMiddleware(cfg.JWT.Secret), authHandler.Me)
		}

		// 知识库路由（需要授权）
		kbs := v1.Group("/kbs", handler.AuthMiddleware(cfg.JWT.Secret))
		{
			kbs.GET("", kbHandler.ListKBs)
			kbs.POST("", kbHandler.CreateKB)

			// :id 路由必须放在前面 - 最具体的带有额外路径段的路由
			kbs.GET("/:id", kbHandler.GetKB)
			kbs.PUT("/:id", kbHandler.UpdateKB)
			kbs.DELETE("/:id", kbHandler.DeleteKB)

			// 知识库下的文档路由
			docs := kbs.Group("/:id/docs")
			{
				docs.GET("", docHandler.ListDocuments)
				docs.POST("", docHandler.UploadDocument)
			}

			// 知识库下的聊天路由
			chat := kbs.Group("/:id/chat")
			{
				chat.POST("", chatHandler.Chat)
			}

			// 知识库下的会话路由
			sessions := kbs.Group("/:id/sessions")
			{
				sessions.GET("", chatHandler.ListSessions)
				sessions.GET("/:session_id/messages", chatHandler.GetSessionMessages)
				sessions.DELETE("/:session_id", chatHandler.DeleteSession)
			}

			// 知识库下的搜索路由
			search := kbs.Group("/:id/search")
			{
				search.GET("", searchHandler.Search)
				search.POST("/hybrid", searchHandler.HybridSearch)
				search.POST("/graph", searchHandler.GraphSearch)
			}
		}

		// 文档路由（需要授权）- 独立文档操作
		docs := v1.Group("/docs", handler.AuthMiddleware(cfg.JWT.Secret))
		{
			docs.GET("/:id", docHandler.GetDocument)
			docs.GET("/:id/content", docHandler.GetDocumentContent)
			docs.DELETE("/:id", docHandler.DeleteDocument)
		}

		// 会话路由（需要授权）- 独立会话操作
		sessions := v1.Group("/sessions", handler.AuthMiddleware(cfg.JWT.Secret))
		{
			sessions.GET("/:id/messages", chatHandler.GetSessionMessages)
			sessions.DELETE("/:id", chatHandler.DeleteSession)
		}

		// 全局聊天路由（需要授权）- 跨知识库聊天
		v1.POST("/chat", handler.AuthMiddleware(cfg.JWT.Secret), globalChatHandler.Chat)
	}

	// 创建 HTTP 服务器
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 启动 HTTP 服务器（在 goroutine 中）
	go func() {
		logger.Info("starting HTTP server", zap.String("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server failed", zap.Error(err))
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	// 优雅关闭（带超时）
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("server forced to shutdown", zap.Error(err))
	}

	log.Println("myRAG server stopped")
}
