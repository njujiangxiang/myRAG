package rag

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"myrag/internal/embedding"
	"myrag/internal/qdrant"
)

// VectorRAG 实现标准向量相似度搜索
type VectorRAG struct {
	qdrantClient    *qdrant.Client
	embeddingClient *embedding.Client
}

// NewVectorRAG 创建一个新的向量 RAG 策略
func NewVectorRAG(qdrantClient *qdrant.Client, embeddingClient *embedding.Client) *VectorRAG {
	return &VectorRAG{
		qdrantClient:    qdrantClient,
		embeddingClient: embeddingClient,
	}
}

// GetType 返回 RAG 类型
func (v *VectorRAG) GetType() RAGType {
	return RAGTypeVector
}

// Search 执行向量相似度搜索
func (v *VectorRAG) Search(ctx context.Context, query string, kbID, tenantID uuid.UUID, limit int) ([]SearchResult, error) {
	// 生成查询的嵌入向量
	queryVector, err := v.embeddingClient.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("生成嵌入向量失败：%w", err)
	}

	// 在 Qdrant 中搜索
	results, err := v.qdrantClient.Search(ctx, qdrant.SearchRequest{
		QueryVector: queryVector,
		TenantID:    tenantID,
		KBID:        kbID,
		Limit:       limit,
	})
	if err != nil {
		return nil, fmt.Errorf("搜索失败：%w", err)
	}

	// 转换为 SearchResult
	searchResults := make([]SearchResult, len(results))
	for i, r := range results {
		searchResults[i] = SearchResult{
			DocumentID: r.DocumentID,
			Content:    r.Content,
			Score:      r.Score,
			Metadata:   r.Metadata,
		}
	}

	return searchResults, nil
}

// IndexDocument 处理和索引文档
func (v *VectorRAG) IndexDocument(ctx context.Context, docID, kbID, tenantID uuid.UUID, content string, mimeType string) error {
	// 将内容分割为块
	chunks := v.chunkContent(content, mimeType)

	// 为所有块生成嵌入向量
	chunksWithEmbeddings := make([]qdrant.Chunk, len(chunks))
	for i, chunk := range chunks {
		// 生成嵌入向量
		embedding, err := v.embeddingClient.GenerateEmbedding(ctx, chunk.Content)
		if err != nil {
			return fmt.Errorf("为块生成嵌入向量失败：%w", err)
		}

		chunksWithEmbeddings[i] = qdrant.Chunk{
			ID:         chunk.ID,
			DocumentID: docID,
			TenantID:   tenantID,
			KBID:       kbID,
			Content:    chunk.Content,
			ChunkIndex: chunk.Index,
			Embedding:  embedding,
			Metadata:   chunk.Metadata,
		}
	}

	// 一次性上传所有块
	return v.qdrantClient.UpsertChunks(ctx, chunksWithEmbeddings)
}

// DeleteDocument 从向量索引中删除文档
func (v *VectorRAG) DeleteDocument(ctx context.Context, docID, kbID, tenantID uuid.UUID) error {
	return v.qdrantClient.DeleteByDocumentID(ctx, tenantID, kbID, docID)
}

// chunkContent 将内容分割为可管理的块
// 这是简化版本 - 应与解析器逻辑保持一致
func (v *VectorRAG) chunkContent(content, mimeType string) []TextChunk {
	const maxChunkSize = 500
	const overlap = 50

	var chunks []TextChunk
	runes := []rune(content)

	for i := 0; i < len(runes); i += maxChunkSize - overlap {
		end := i + maxChunkSize
		if end > len(runes) {
			end = len(runes)
		}

		chunkContent := string(runes[i:end])
		chunks = append(chunks, TextChunk{
			ID:       uuid.New(),
			Content:  chunkContent,
			Index:    len(chunks),
			Metadata: map[string]any{"mime_type": mimeType},
		})

		if end >= len(runes) {
			break
		}
	}

	return chunks
}
