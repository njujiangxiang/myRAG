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

// SearchHandler handles search requests
type SearchHandler struct {
	docRepo   *models.DocumentRepository
	qdrant    *qdrant.Client
	embedding *embedding.Client
}

// NewSearchHandler creates a new search handler
func NewSearchHandler(docRepo *models.DocumentRepository, qdrantClient *qdrant.Client, embeddingClient *embedding.Client) *SearchHandler {
	return &SearchHandler{
		docRepo:   docRepo,
		qdrant:    qdrantClient,
		embedding: embeddingClient,
	}
}

// SearchRequest represents a search request
type SearchRequest struct {
	Query string `form:"query" binding:"required"`
	Limit int    `form:"limit" binding:"omitempty,min=1,max=100"`
}

// SearchResult represents a search result
type SearchResult struct {
	DocumentID uuid.UUID `json:"document_id"`
	Filename   string    `json:"filename"`
	Content    string    `json:"content"`
	Score      float32   `json:"score"`
	Metadata   ChunkMeta `json:"metadata,omitempty"`
}

// ChunkMeta represents chunk metadata
type ChunkMeta struct {
	ChunkIndex int       `json:"chunk_index"`
	Page       *int      `json:"page,omitempty"`
	Source     string    `json:"source"`
}

// SearchResponse represents a search response
type SearchResponse struct {
	Results    []SearchResult `json:"results"`
	Total      int            `json:"total"`
	DurationMs int64          `json:"duration_ms"`
}

// Search handles searching documents in a knowledge base
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

	// Get tenant ID from context
	tenantID, ok := GetTenantID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant context missing"})
		return
	}

	// Generate embedding for the query
	queryVector, err := h.embedding.GenerateEmbedding(c.Request.Context(), req.Query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate query embedding"})
		return
	}

	// Search in Qdrant
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

	// Convert results
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

// HybridSearchRequest represents a hybrid search request (vector + keyword)
type HybridSearchRequest struct {
	Query     string  `json:"query" binding:"required"`
	Limit     int     `json:"limit" binding:"omitempty,min=1,max=100"`
	VectorW   float64 `json:"vector_weight" binding:"omitempty,min=0,max=1"`
	KeywordW  float64 `json:"keyword_weight" binding:"omitempty,min=0,max=1"`
	WithGraph bool    `json:"with_graph" binding:"omitempty"`
}

// HybridSearch handles hybrid search (vector + keyword + graph)
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

	// Default weights
	if req.VectorW == 0 && req.KeywordW == 0 {
		req.VectorW = 0.6
		req.KeywordW = 0.4
	}

	// Get tenant ID from context
	tenantID, ok := GetTenantID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant context missing"})
		return
	}

	startTime := time.Now()

	// Generate embedding for the query
	queryVector, err := h.embedding.GenerateEmbedding(c.Request.Context(), req.Query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate query embedding"})
		return
	}

	// Vector search in Qdrant
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

	// For now, return vector results with weighted scores
	// TODO: Implement BM25 keyword search and fuse results
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

// GraphSearchRequest represents a graph search request
type GraphSearchRequest struct {
	Query string `json:"query" binding:"required"`
}

// GraphSearchResult represents a graph search result
type GraphSearchResult struct {
	EntityType    string                 `json:"entity_type"`
	EntityName    string                 `json:"entity_name"`
	Relationships []RelationshipResult   `json:"relationships,omitempty"`
	CommunitySummary *string             `json:"community_summary,omitempty"`
	Score         float32                `json:"score"`
	Metadata      map[string]any         `json:"metadata,omitempty"`
}

// RelationshipResult represents a relationship in graph search
type RelationshipResult struct {
	Source      string `json:"source"`
	Target      string `json:"target"`
	Description string `json:"description"`
}

// GraphSearch handles graph-based search
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

	// Get tenant ID from context
	tenantID, ok := GetTenantID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant context missing"})
		return
	}

	// Graph search is a v1.1 features
	// For now, return vector search results as fallback
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

	// Convert to graph search results (placeholder)
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
