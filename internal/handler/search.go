package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"myrag/internal/embedding"
	"myrag/internal/models"
	"myrag/internal/qdrant"
)

// SearchHandler 处理搜索请求
type SearchHandler struct {
	docRepo   *models.DocumentRepository
	qdrant    *qdrant.Client
	embedding *embedding.Client
}

// NewSearchHandler 创建一个新的搜索处理器
func NewSearchHandler(docRepo *models.DocumentRepository, qdrantClient *qdrant.Client, embeddingClient *embedding.Client) *SearchHandler {
	return &SearchHandler{
		docRepo:   docRepo,
		qdrant:    qdrantClient,
		embedding: embeddingClient,
	}
}

// SearchRequest 表示搜索请求
type SearchRequest struct {
	Query string `form:"query" binding:"required"`
	Limit int    `form:"limit" binding:"omitempty,min=1,max=100"`
}

// SearchResult 表示搜索结果
type SearchResult struct {
	DocumentID uuid.UUID `json:"document_id"`
	Filename   string    `json:"filename"`
	Content    string    `json:"content"`
	Score      float32   `json:"score"`
	Metadata   ChunkMeta `json:"metadata,omitempty"`
}

// ChunkMeta 表示片段元数据
type ChunkMeta struct {
	ChunkIndex int       `json:"chunk_index"`
	Page       *int      `json:"page,omitempty"`
	Source     string    `json:"source"`
}

// SearchResponse 表示搜索响应
type SearchResponse struct {
	Results    []SearchResult `json:"results"`
	Total      int            `json:"total"`
	DurationMs int64          `json:"duration_ms"`
}

// Search 处理知识库文档搜索
// GET /api/v1/kbs/:id/search
func (h *SearchHandler) Search(c *gin.Context) {
	kbIDStr := c.Param("id")
	kbID, err := uuid.Parse(kbIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid knowledge base ID"})
		return
	}

	var req SearchRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query is required"})
		return
	}

	if req.Limit == 0 {
		req.Limit = 10
	}

	startTime := time.Now()

	// 从上下文中获取租户 ID
	tenantID, ok := GetTenantID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant context missing"})
		return
	}

	// 生成查询的嵌入向量
	queryVector, err := h.embedding.GenerateEmbedding(c.Request.Context(), req.Query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate query embedding"})
		return
	}

	// 在 Qdrant 中搜索
	results, err := h.qdrant.Search(c.Request.Context(), qdrant.SearchRequest{
		QueryVector: queryVector,
		TenantID:    tenantID,
		KBID:        kbID,
		Limit:       req.Limit,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed"})
		return
	}

	// 转换结果
	searchResults := make([]SearchResult, len(results))
	for i, r := range results {
		searchResults[i] = SearchResult{
			DocumentID: r.DocumentID,
			Content:    r.Content,
			Score:      r.Score,
			Metadata: ChunkMeta{
				ChunkIndex: r.ChunkIndex,
				Source:     "vector",
			},
		}
	}

	c.JSON(http.StatusOK, SearchResponse{
		Results:    searchResults,
		Total:      len(searchResults),
		DurationMs: time.Since(startTime).Milliseconds(),
	})
}

// HybridSearchRequest 表示混合搜索请求（向量 + 关键词）
type HybridSearchRequest struct {
	Query     string  `json:"query" binding:"required"`
	Limit     int     `json:"limit" binding:"omitempty,min=1,max=100"`
	VectorW   float64 `json:"vector_weight" binding:"omitempty,min=0,max=1"`
	KeywordW  float64 `json:"keyword_weight" binding:"omitempty,min=0,max=1"`
	WithGraph bool    `json:"with_graph" binding:"omitempty"`
}

// HybridSearch 处理混合搜索（向量 + 关键词 + 图谱）
// POST /api/v1/kbs/:id/search/hybrid
func (h *SearchHandler) HybridSearch(c *gin.Context) {
	kbIDStr := c.Param("id")
	kbID, err := uuid.Parse(kbIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid knowledge base ID"})
		return
	}

	var req HybridSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 默认权重
	if req.VectorW == 0 && req.KeywordW == 0 {
		req.VectorW = 0.6
		req.KeywordW = 0.4
	}

	// 从上下文中获取租户 ID
	tenantID, ok := GetTenantID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant context missing"})
		return
	}

	startTime := time.Now()

	// 生成查询的嵌入向量
	queryVector, err := h.embedding.GenerateEmbedding(c.Request.Context(), req.Query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate query embedding"})
		return
	}

	// 在 Qdrant 中进行向量搜索
	vectorResults, err := h.qdrant.Search(c.Request.Context(), qdrant.SearchRequest{
		QueryVector: queryVector,
		TenantID:    tenantID,
		KBID:        kbID,
		Limit:       req.Limit,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "vector search failed"})
		return
	}

	// 暂时返回带有权重分数的向量结果
	// TODO: 实现 BM25 关键词搜索并融合结果
	searchResults := make([]SearchResult, len(vectorResults))
	for i, r := range vectorResults {
		searchResults[i] = SearchResult{
			DocumentID: r.DocumentID,
			Content:    r.Content,
			Score:      r.Score * float32(req.VectorW),
			Metadata: ChunkMeta{
				ChunkIndex: r.ChunkIndex,
				Source:     "hybrid",
			},
		}
	}

	c.JSON(http.StatusOK, SearchResponse{
		Results:    searchResults,
		Total:      len(searchResults),
		DurationMs: time.Since(startTime).Milliseconds(),
	})
}

// GraphSearchRequest 表示图谱搜索请求
type GraphSearchRequest struct {
	Query string `json:"query" binding:"required"`
}

// GraphSearchResult 表示图谱搜索结果
type GraphSearchResult struct {
	EntityType    string                 `json:"entity_type"`
	EntityName    string                 `json:"entity_name"`
	Relationships []RelationshipResult   `json:"relationships,omitempty"`
	CommunitySummary *string             `json:"community_summary,omitempty"`
	Score         float32                `json:"score"`
	Metadata      map[string]any         `json:"metadata,omitempty"`
}

// RelationshipResult 表示图谱搜索中的关系
type RelationshipResult struct {
	Source      string `json:"source"`
	Target      string `json:"target"`
	Description string `json:"description"`
}

// GraphSearch 处理基于图谱的搜索
// POST /api/v1/kbs/:id/search/graph
func (h *SearchHandler) GraphSearch(c *gin.Context) {
	kbIDStr := c.Param("id")
	kbID, err := uuid.Parse(kbIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid knowledge base ID"})
		return
	}

	var req GraphSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 从上下文中获取租户 ID
	tenantID, ok := GetTenantID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant context missing"})
		return
	}

	// 图谱搜索是 v1.1 功能
	// 暂时返回向量搜索结果作为降级
	queryVector, err := h.embedding.GenerateEmbedding(c.Request.Context(), req.Query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate query embedding"})
		return
	}

	results, err := h.qdrant.Search(c.Request.Context(), qdrant.SearchRequest{
		QueryVector: queryVector,
		TenantID:    tenantID,
		KBID:        kbID,
		Limit:       10,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed"})
		return
	}

	// 转换为图谱搜索结果（占位符）
	graphResults := make([]GraphSearchResult, len(results))
	for i, r := range results {
		graphResults[i] = GraphSearchResult{
			EntityType: "chunk",
			EntityName: r.Content[:min(100, len(r.Content))],
			Score:      r.Score,
			Metadata: map[string]any{
				"document_id": r.DocumentID.String(),
				"chunk_index": r.ChunkIndex,
			},
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"results": graphResults,
		"note":    "Full graph search coming in v1.1 - currently returning vector search results",
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
