package rag

import (
	"context"
	"fmt"
	"os"
	"sort"
	"sync"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/index/scorch"
	"github.com/google/uuid"

	"myrag/internal/embedding"
	"myrag/internal/qdrant"
)

// HybridRAG 实现向量 + 关键词混合搜索与 RRF 融合
type HybridRAG struct {
	qdrantClient    *qdrant.Client
	embeddingClient *embedding.Client
	bm25Index       bleve.Index
	bm25Mutex       sync.RWMutex
	indexPath       string
}

// NewHybridRAG 创建一个新的混合 RAG 策略
func NewHybridRAG(qdrantClient *qdrant.Client, embeddingClient *embedding.Client, indexPath string) *HybridRAG {
	h := &HybridRAG{
		qdrantClient:    qdrantClient,
		embeddingClient: embeddingClient,
		indexPath:       indexPath,
	}

	// 初始化或打开 BM25 索引
	if err := h.initBM25Index(); err != nil {
		// 记录错误但继续（BM25 将被禁用）
		fmt.Printf("警告：初始化 BM25 索引失败：%v\n", err)
	}

	return h
}

// initBM25Index 初始化或打开 BM25 索引
func (h *HybridRAG) initBM25Index() error {
	h.bm25Mutex.Lock()
	defer h.bm25Mutex.Unlock()

	// 确保索引目录存在
	if err := os.MkdirAll(h.indexPath, 0755); err != nil {
		return fmt.Errorf("创建索引目录失败：%w", err)
	}

	// 尝试打开现有索引
	index, err := bleve.Open(h.indexPath)
	if err == nil {
		h.bm25Index = index
		return nil
	}

	// 使用 BM25 相似度创建新索引
	indexMapping := bleve.NewIndexMapping()
	indexMapping.TypeField = "type"
	indexMapping.DefaultAnalyzer = "en"

	// 配置 BM25 相似度
	indexMapping.StoreDynamic = true
	indexMapping.DefaultMapping = bleve.NewDocumentMapping()

	index, err = bleve.NewUsing(h.indexPath, indexMapping, scorch.Name, scorch.Name, nil)
	if err != nil {
		return fmt.Errorf("创建 BM25 索引失败：%w", err)
	}

	h.bm25Index = index
	return nil
}

// GetType 返回 RAG 类型
func (h *HybridRAG) GetType() RAGType {
	return RAGTypeHybrid
}

// Search 执行混合搜索与 RRF 融合
func (h *HybridRAG) Search(ctx context.Context, query string, kbID, tenantID uuid.UUID, limit int) ([]SearchResult, error) {
	// 向量搜索
	vectorResults, err := h.vectorSearch(ctx, query, kbID, tenantID, limit*2)
	if err != nil {
		return nil, fmt.Errorf("向量搜索失败：%w", err)
	}

	// 关键词搜索 (BM25)
	keywordResults, err := h.keywordSearch(ctx, query, kbID, tenantID, limit*2)
	if err != nil {
		return nil, fmt.Errorf("关键词搜索失败：%w", err)
	}

	// RRF 融合：倒数排名融合
	fusedResults := h.rrfFusion(vectorResults, keywordResults, limit)

	return fusedResults, nil
}

// vectorSearch 执行标准向量相似度搜索
func (h *HybridRAG) vectorSearch(ctx context.Context, query string, kbID, tenantID uuid.UUID, limit int) ([]SearchResult, error) {
	queryVector, err := h.embeddingClient.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, err
	}

	results, err := h.qdrantClient.Search(ctx, qdrant.SearchRequest{
		QueryVector: queryVector,
		TenantID:    tenantID,
		KBID:        kbID,
		Limit:       limit,
	})
	if err != nil {
		return nil, err
	}

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

// keywordSearch 执行 BM25 关键词搜索
func (h *HybridRAG) keywordSearch(ctx context.Context, query string, kbID, tenantID uuid.UUID, limit int) ([]SearchResult, error) {
	h.bm25Mutex.RLock()
	defer h.bm25Mutex.RUnlock()

	if h.bm25Index == nil {
		return []SearchResult{}, nil
	}

	// 构建带租户和知识库过滤的查询
	queryStr := fmt.Sprintf("+tenant_id:%s +kb_id:%s %s", tenantID.String(), kbID.String(), query)
	bleveQuery := bleve.NewQueryStringQuery(queryStr)

	searchRequest := bleve.NewSearchRequest(bleveQuery)
	searchRequest.Size = limit
	searchRequest.Fields = []string{"content", "document_id", "chunk_index"}

	searchResult, err := h.bm25Index.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("BM25 搜索失败：%w", err)
	}

	results := make([]SearchResult, len(searchResult.Hits))
	for i, hit := range searchResult.Hits {
		docID, _ := uuid.Parse(hit.Fields["document_id"].(string))
		content := hit.Fields["content"].(string)

		results[i] = SearchResult{
			DocumentID: docID,
			Content:    content,
			Score:      float32(hit.Score),
			Metadata: map[string]any{
				"chunk_index": int(hit.Fields["chunk_index"].(float64)),
			},
		}
	}

	return results, nil
}

// rrfFusion 应用倒数排名融合来合并结果
// RRF 公式：score = sum(1 / (k + rank)) 对于每个结果
// 默认 k=60
func (h *HybridRAG) rrfFusion(vectorResults, keywordResults []SearchResult, limit int) []SearchResult {
	const rrfK = 60

	// 映射以累积 RRF 分数
	rrfScores := make(map[uuid.UUID]float64)
	resultMap := make(map[uuid.UUID]SearchResult)

	// 计算向量结果分数
	for rank, r := range vectorResults {
		score := 1.0 / float64(rrfK+rank+1)
		rrfScores[r.DocumentID] += score
		resultMap[r.DocumentID] = r
	}

	// 计算关键词结果分数，权重更高
	for rank, r := range keywordResults {
		score := 2.0 / float64(rrfK+rank+1) // 关键词结果获得 2 倍权重
		rrfScores[r.DocumentID] += score
		if _, exists := resultMap[r.DocumentID]; !exists {
			resultMap[r.DocumentID] = r
		}
	}

	// 按 RRF 分数排序
	type scoredResult struct {
		result SearchResult
		score  float64
	}

	var scored []scoredResult
	for id, score := range rrfScores {
		if result, ok := resultMap[id]; ok {
			scored = append(scored, scoredResult{result: result, score: score})
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// 返回前 N 个结果
	finalResults := make([]SearchResult, 0, limit)
	for i := 0; i < len(scored) && i < limit; i++ {
		finalResults = append(finalResults, scored[i].result)
	}

	return finalResults
}

// IndexDocument 处理文档并用向量和关键词索引建立索引
func (h *HybridRAG) IndexDocument(ctx context.Context, docID, kbID, tenantID uuid.UUID, content string, mimeType string) error {
	// 索引向量
	chunks := h.chunkContent(content, mimeType)

	// 为所有块生成嵌入向量
	chunksWithEmbeddings := make([]qdrant.Chunk, len(chunks))
	for i, chunk := range chunks {
		embedding, err := h.embeddingClient.GenerateEmbedding(ctx, chunk.Content)
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

	// 一次性上传所有块到 Qdrant
	if err := h.qdrantClient.UpsertChunks(ctx, chunksWithEmbeddings); err != nil {
		return err
	}

	// 索引到 BM25
	if err := h.indexBM25(docID, kbID, tenantID, chunks); err != nil {
		return fmt.Errorf("索引 BM25 失败：%w", err)
	}

	return nil
}

// indexBM25 将块索引到 BM25 索引
func (h *HybridRAG) indexBM25(docID, kbID, tenantID uuid.UUID, chunks []TextChunk) error {
	h.bm25Mutex.Lock()
	defer h.bm25Mutex.Unlock()

	if h.bm25Index == nil {
		return nil // BM25 禁用，跳过索引
	}

	for _, chunk := range chunks {
		doc := map[string]interface{}{
			"type":        "chunk",
			"document_id": docID.String(),
			"kb_id":       kbID.String(),
			"tenant_id":   tenantID.String(),
			"content":     chunk.Content,
			"chunk_index": chunk.Index,
		}

		chunkID := fmt.Sprintf("%s_%d", docID.String(), chunk.Index)
		if err := h.bm25Index.Index(chunkID, doc); err != nil {
			return fmt.Errorf("索引块 %s 失败：%w", chunkID, err)
		}
	}

	return nil
}

// DeleteDocument 从所有索引中删除文档
func (h *HybridRAG) DeleteDocument(ctx context.Context, docID, kbID, tenantID uuid.UUID) error {
	// 删除向量
	if err := h.qdrantClient.DeleteByDocumentID(ctx, tenantID, kbID, docID); err != nil {
		return err
	}

	// 从 BM25 索引删除
	if err := h.deleteBM25(docID); err != nil {
		return fmt.Errorf("删除 BM25 失败：%w", err)
	}

	return nil
}

// deleteBM25 从 BM25 索引删除文档
func (h *HybridRAG) deleteBM25(docID uuid.UUID) error {
	h.bm25Mutex.Lock()
	defer h.bm25Mutex.Unlock()

	if h.bm25Index == nil {
		return nil
	}

	// 删除该文档的所有块
	// 需要先查询获取所有块 ID
	queryStr := fmt.Sprintf("document_id:%s", docID.String())
	bleveQuery := bleve.NewQueryStringQuery(queryStr)

	searchRequest := bleve.NewSearchRequest(bleveQuery)
	searchRequest.Size = 1000 // 获取所有块
	searchRequest.Fields = []string{"chunk_index"}

	searchResult, err := h.bm25Index.Search(searchRequest)
	if err != nil {
		return fmt.Errorf("搜索删除失败：%w", err)
	}

	// 删除每个块
	for _, hit := range searchResult.Hits {
		if err := h.bm25Index.Delete(hit.ID); err != nil {
			return fmt.Errorf("删除块 %s 失败：%w", hit.ID, err)
		}
	}

	return nil
}

func (h *HybridRAG) chunkContent(content, mimeType string) []TextChunk {
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
