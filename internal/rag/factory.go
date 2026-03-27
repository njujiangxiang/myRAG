package rag

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"

	"myrag/internal/embedding"
	"myrag/internal/models"
	"myrag/internal/qdrant"
	"myrag/internal/rerank"
)

// Factory 创建和管理 RAG 策略实例
type Factory struct {
	vectorRAG   *VectorRAG
	graphRAG    *GraphRAG
	hybridRAG   *HybridRAG
	llmClient   *LLMClient
	strategies  map[RAGType]RAGStrategy
	mu          sync.RWMutex
}

// FactoryConfig 持有创建 RAG 策略的配置
type FactoryConfig struct {
	QdrantClient    *qdrant.Client    // Qdrant 向量数据库客户端
	EmbeddingClient *embedding.Client // 嵌入生成客户端
	LLMAPIKey       string            // LLM API 密钥
	LLMModel        string            // LLM 模型名称
	LLMProvider     string            // LLM 提供商（openai/anthropic）
	BM25IndexPath   string            // BM25 索引路径
	Rerank          *RerankConfig     // 可选的重排序配置
}

// RerankConfig 持有 BGE 自托管重排序服务的配置
type RerankConfig struct {
	Enabled    bool   // 是否启用重排序
	BaseURL    string // BGE 服务地址
	Model      string // 模型名称
	TopK       int    // 返回结果数量
	Candidates int    // 候选结果数量
}

// NewFactory 创建新的 RAG 策略工厂
func NewFactory(cfg FactoryConfig) *Factory {
	// 为 Graph RAG 创建 LLM 客户端
	llmClient := NewLLMClient(cfg.LLMAPIKey, cfg.LLMModel, cfg.LLMProvider)

	vectorRAG := NewVectorRAG(cfg.QdrantClient, cfg.EmbeddingClient)
	graphRAG := NewGraphRAG(cfg.QdrantClient, cfg.EmbeddingClient, llmClient)
	hybridRAG := NewHybridRAG(cfg.QdrantClient, cfg.EmbeddingClient, cfg.BM25IndexPath)

	factory := &Factory{
		vectorRAG:  vectorRAG,
		graphRAG:   graphRAG,
		hybridRAG:  hybridRAG,
		llmClient:  llmClient,
		strategies: make(map[RAGType]RAGStrategy),
	}

	// 注册默认策略
	factory.strategies[RAGTypeVector] = vectorRAG
	factory.strategies[RAGTypeGraph] = graphRAG
	factory.strategies[RAGTypeHybrid] = hybridRAG

	// 如果启用了重排序，注册重排序策略
	if cfg.Rerank != nil && cfg.Rerank.Enabled {
		rerankClient := rerank.New(rerank.Config{
			BaseURL: cfg.Rerank.BaseURL,
			Model:   cfg.Rerank.Model,
			TopK:    cfg.Rerank.TopK,
		})
		// 默认情况下，重排序策略包装 HybridRAG
		rerankRAG := NewRerankRAG(hybridRAG, rerankClient, cfg.Rerank.Candidates, cfg.Rerank.TopK)
		factory.strategies[RAGTypeRerank] = rerankRAG
	}

	return factory
}

// GetStrategy 返回指定类型的 RAG 策略
func (f *Factory) GetStrategy(ragType RAGType) (RAGStrategy, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	strategy, exists := f.strategies[ragType]
	if !exists {
		return nil, fmt.Errorf("不支持的 RAG 类型：%s", ragType)
	}

	return strategy, nil
}

// KBRepository 定义知识库访问的最小接口
type KBRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*models.KnowledgeBase, error)
}

// GetStrategyByKB 返回知识库适用的 RAG 策略
func (f *Factory) GetStrategyByKB(ctx context.Context, kbID, tenantID uuid.UUID, kbRepo KBRepository) (RAGStrategy, error) {
	kb, err := kbRepo.GetByID(ctx, kbID)
	if err != nil {
		return nil, fmt.Errorf("获取知识库失败：%w", err)
	}

	ragType := RAGType(kb.RAGType)
	if ragType == "" {
		ragType = RAGTypeVector // 默认为向量检索
	}

	return f.GetStrategy(ragType)
}

// ListStrategies 返回所有已注册的策略类型
func (f *Factory) ListStrategies() []RAGType {
	f.mu.RLock()
	defer f.mu.RUnlock()

	types := make([]RAGType, 0, len(f.strategies))
	for t := range f.strategies {
		types = append(types, t)
	}

	return types
}
