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

	// Initialize logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load configuration", zap.Error(err))
	}
	logger.Info("configuration loaded", zap.String("env", cfg.Server.Env))

	// Initialize database
	db, err := database.New(cfg.Database.URL, logger)
	if err != nil {
		logger.Fatal("failed to initialize database", zap.Error(err))
	}
	defer db.Close()

	// Initialize Qdrant client
	qdrantClient, err := qdrant.New(cfg.Qdrant.URL, cfg.Qdrant.APIKey, "documents", logger)
	if err != nil {
		logger.Fatal("failed to initialize Qdrant", zap.Error(err))
	}
	defer qdrantClient.Close()

	// Initialize MinIO client
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

	// Initialize NATS client
	natsClient, err := nats.New(cfg.NATS.URL, logger)
	if err != nil {
		logger.Fatal("failed to initialize NATS", zap.Error(err))
	}
	defer natsClient.Close()

	// Ensure NATS stream exists
	if err := natsClient.EnsureStream(context.Background(), "documents"); err != nil {
		logger.Warn("failed to ensure NATS stream", zap.Error(err))
	}

	// Initialize repositories
	userRepo := models.NewUserRepository(db.DB)
	kbRepo := models.NewKnowledgeBaseRepository(db.DB)
	docRepo := models.NewDocumentRepository(db.DB)
	sessionRepo := models.NewChatSessionRepository(db.DB)
	messageRepo := models.NewMessageRepository(db.DB)

	// Initialize embedding client
	embeddingConfig := embedding.DefaultConfig()
	embeddingClient := embedding.New(embeddingConfig)
	logger.Info("embedding client initialized", zap.String("model", embeddingClient.GetModel()))

	// Initialize RAG factory
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

	// Initialize handlers
	authHandler := handler.NewAuthHandler(userRepo, cfg.JWT.Secret)
	kbHandler := handler.NewKnowledgeBaseHandler(kbRepo)
	docHandler := handler.NewDocumentHandler(docRepo, kbRepo, minioClient, natsClient, qdrantClient)
	chatHandler := handler.NewChatHandler(sessionRepo, messageRepo, kbRepo, qdrantClient, embeddingClient, ragFactory, cfg.LLM.APIKey, cfg.LLM.Model, cfg.LLM.Provider)
	globalChatHandler := handler.NewGlobalChatHandler(sessionRepo, messageRepo, kbRepo, qdrantClient, embeddingClient, cfg.LLM.APIKey, cfg.LLM.Model, cfg.LLM.Provider)
	searchHandler := handler.NewSearchHandler(docRepo, qdrantClient, embeddingClient)

	// Initialize and start document worker
	docWorker, err := worker.NewWorker(docRepo, minioClient, qdrantClient, embeddingClient, natsClient.JetStream)
	if err != nil {
		logger.Fatal("failed to initialize document worker", zap.Error(err))
	}
	go docWorker.Start(context.Background())

	// Setup Gin router
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().UTC(),
		})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Auth routes (no auth required)
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/logout", authHandler.Logout)
			auth.GET("/me", handler.AuthMiddleware(cfg.JWT.Secret), authHandler.Me)
		}

		// Knowledge Base routes (auth required)
		kbs := v1.Group("/kbs", handler.AuthMiddleware(cfg.JWT.Secret))
		{
			kbs.GET("", kbHandler.ListKBs)
			kbs.POST("", kbHandler.CreateKB)

			// :id routes must be first - most specific patterns with additional path segments
			kbs.GET("/:id", kbHandler.GetKB)
			kbs.PUT("/:id", kbHandler.UpdateKB)
			kbs.DELETE("/:id", kbHandler.DeleteKB)

			// Document routes under KB
			docs := kbs.Group("/:id/docs")
			{
				docs.GET("", docHandler.ListDocuments)
				docs.POST("", docHandler.UploadDocument)
			}

			// Chat routes under KB
			chat := kbs.Group("/:id/chat")
			{
				chat.POST("", chatHandler.Chat)
			}

			// Session routes under KB
			sessions := kbs.Group("/:id/sessions")
			{
				sessions.GET("", chatHandler.ListSessions)
				sessions.GET("/:session_id/messages", chatHandler.GetSessionMessages)
				sessions.DELETE("/:session_id", chatHandler.DeleteSession)
			}

			// Search routes under KB
			search := kbs.Group("/:id/search")
			{
				search.GET("", searchHandler.Search)
				search.POST("/hybrid", searchHandler.HybridSearch)
				search.POST("/graph", searchHandler.GraphSearch)
			}
		}

		// Document routes (auth required) - standalone document operations
		docs := v1.Group("/docs", handler.AuthMiddleware(cfg.JWT.Secret))
		{
			docs.GET("/:id", docHandler.GetDocument)
			docs.GET("/:id/content", docHandler.GetDocumentContent)
			docs.DELETE("/:id", docHandler.DeleteDocument)
		}

		// Session routes (auth required) - standalone session operations
		sessions := v1.Group("/sessions", handler.AuthMiddleware(cfg.JWT.Secret))
		{
			sessions.GET("/:id/messages", chatHandler.GetSessionMessages)
			sessions.DELETE("/:id", chatHandler.DeleteSession)
		}

		// Global chat route (auth required) - cross-KB chat
		v1.POST("/chat", handler.AuthMiddleware(cfg.JWT.Secret), globalChatHandler.Chat)
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info("starting HTTP server", zap.String("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server failed", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("server forced to shutdown", zap.Error(err))
	}

	log.Println("myRAG server stopped")
}
