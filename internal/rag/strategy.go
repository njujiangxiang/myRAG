package rag

import (
	"context"

	"github.com/google/uuid"
)

// SearchResult 表示 RAG 搜索的结果
type SearchResult struct {
	DocumentID uuid.UUID `json:"document_id"`
	Content    string    `json:"content"`
	Score      float32   `json:"score"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// RAGType 定义 RAG 策略类型
type RAGType string

const (
	RAGTypeVector  RAGType = "vector"  // 标准向量相似度搜索
	RAGTypeGraph   RAGType = "graph"   // 知识图谱增强搜索
	RAGTypeHybrid  RAGType = "hybrid"  // 向量 + 关键词混合搜索
	RAGTypeKeyword RAGType = "keyword" // 纯关键词搜索 (BM25)
	RAGTypeRerank  RAGType = "rerank"  // Cross-Encoder 重排序搜索
)

// AllRAGTypes 返回所有支持的 RAG 类型
func AllRAGTypes() []RAGType {
	return []RAGType{RAGTypeVector, RAGTypeGraph, RAGTypeHybrid, RAGTypeKeyword, RAGTypeRerank}
}

// IsValidRAGType 检查字符串是否为有效的 RAG 类型
func IsValidRAGType(s string) bool {
	for _, t := range AllRAGTypes() {
		if string(t) == s {
			return true
		}
	}
	return false
}

// TextChunk 表示用于内部处理的文本块
type TextChunk struct {
	ID       uuid.UUID
	Content  string
	Index    int
	Metadata map[string]any
}

// RAGStrategy 定义 RAG 检索的接口
type RAGStrategy interface {
	// GetType 返回 RAG 类型标识符
	GetType() RAGType

	// Search 检索查询相关的文档块
	Search(ctx context.Context, query string, kbID, tenantID uuid.UUID, limit int) ([]SearchResult, error)

	// IndexDocument 处理和索引新文档
	IndexDocument(ctx context.Context, docID, kbID, tenantID uuid.UUID, content string, mimeType string) error

	// DeleteDocument 从索引中删除文档
	DeleteDocument(ctx context.Context, docID, kbID, tenantID uuid.UUID) error
}
