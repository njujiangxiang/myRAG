package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"myrag/internal/models"
)

// KnowledgeBaseHandler handles knowledge base requests
type KnowledgeBaseHandler struct {
	kbRepo *models.KnowledgeBaseRepository
}

// NewKnowledgeBaseHandler creates a new KB handler
func NewKnowledgeBaseHandler(kbRepo *models.KnowledgeBaseRepository) *KnowledgeBaseHandler {
	return &KnowledgeBaseHandler{
		kbRepo: kbRepo,
	}
}

// CreateKBRequest represents a create KB request
type CreateKBRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description,omitempty"`
	RAGType     string  `json:"rag_type,omitempty"` // vector, graph, hybrid, keyword
}

// KBResult represents knowledge base data in response
type KBResult struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	OwnerID     uuid.UUID  `json:"owner_id"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	RAGType     string     `json:"rag_type"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ListKBs handles listing knowledge bases for current tenant
// GET /api/v1/kbs
func (h *KnowledgeBaseHandler) ListKBs(c *gin.Context) {
	tenantID, ok := GetTenantID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant context missing"})
		return
	}

	kbs, err := h.kbRepo.GetByTenant(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list knowledge bases"})
		return
	}

	results := make([]KBResult, len(kbs))
	for i, kb := range kbs {
		results[i] = KBResult{
			ID:          kb.ID,
			TenantID:    kb.TenantID,
			OwnerID:     kb.OwnerID,
			Name:        kb.Name,
			Description: kb.Description,
			CreatedAt:   kb.CreatedAt,
			UpdatedAt:   kb.UpdatedAt,
		}
	}

	c.JSON(http.StatusOK, results)
}

// GetKB handles getting a single knowledge base
// GET /api/v1/kbs/:id
func (h *KnowledgeBaseHandler) GetKB(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid knowledge base ID"})
		return
	}

	// Get tenant ID from context
	tenantID, ok := GetTenantID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant context missing"})
		return
	}

	kb, err := h.kbRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "knowledge base not found"})
		return
	}

	// Verify tenant ownership
	if kb.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	c.JSON(http.StatusOK, KBResult{
		ID:          kb.ID,
		TenantID:    kb.TenantID,
		OwnerID:     kb.OwnerID,
		Name:        kb.Name,
		Description: kb.Description,
		CreatedAt:   kb.CreatedAt,
		UpdatedAt:   kb.UpdatedAt,
	})
}

// CreateKB handles creating a new knowledge base
// POST /api/v1/kbs
func (h *KnowledgeBaseHandler) CreateKB(c *gin.Context) {
	var req CreateKBRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenantID, ok := GetTenantID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant context missing"})
		return
	}

	userID, ok := GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user context missing"})
		return
	}

	kb := &models.KnowledgeBase{
		ID:          uuid.New(),
		TenantID:    tenantID,
		OwnerID:     userID,
		Name:        req.Name,
		Description: req.Description,
		RAGType:     getRAGType(req.RAGType),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := h.kbRepo.Create(c.Request.Context(), kb); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create knowledge base"})
		return
	}

	c.JSON(http.StatusCreated, KBResult{
		ID:          kb.ID,
		TenantID:    kb.TenantID,
		OwnerID:     kb.OwnerID,
		Name:        kb.Name,
		Description: kb.Description,
		CreatedAt:   kb.CreatedAt,
		UpdatedAt:   kb.UpdatedAt,
	})
}

// UpdateKB handles updating a knowledge base
// PUT /api/v1/kbs/:id
func (h *KnowledgeBaseHandler) UpdateKB(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid knowledge base ID"})
		return
	}

	// Get tenant ID from context
	tenantID, ok := GetTenantID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant context missing"})
		return
	}

	var req CreateKBRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	kb, err := h.kbRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "knowledge base not found"})
		return
	}

	// Verify tenant ownership
	if kb.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	kb.Name = req.Name
	kb.Description = req.Description
	kb.RAGType = getRAGType(req.RAGType)
	kb.UpdatedAt = time.Now()

	if err := h.kbRepo.Update(c.Request.Context(), kb); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update knowledge base"})
		return
	}

	c.JSON(http.StatusOK, KBResult{
		ID:          kb.ID,
		TenantID:    kb.TenantID,
		OwnerID:     kb.OwnerID,
		Name:        kb.Name,
		Description: kb.Description,
		CreatedAt:   kb.CreatedAt,
		UpdatedAt:   kb.UpdatedAt,
	})
}

// DeleteKB handles deleting a knowledge base
// DELETE /api/v1/kbs/:id
func (h *KnowledgeBaseHandler) DeleteKB(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid knowledge base ID"})
		return
	}

	// Get tenant ID from context
	tenantID, ok := GetTenantID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant context missing"})
		return
	}

	kb, err := h.kbRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "knowledge base not found"})
		return
	}

	// Verify tenant ownership
	if kb.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if err := h.kbRepo.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete knowledge base"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "knowledge base deleted"})
}

// kbResponse is a helper to format KB response
func kbResponse(kb *models.KnowledgeBase) KBResult {
	return KBResult{
		ID:          kb.ID,
		TenantID:    kb.TenantID,
		OwnerID:     kb.OwnerID,
		Name:        kb.Name,
		Description: kb.Description,
		RAGType:     kb.RAGType,
		CreatedAt:   kb.CreatedAt,
		UpdatedAt:   kb.UpdatedAt,
	}
}

// getRAGType returns a valid RAG type or default
func getRAGType(s string) string {
	if s == "" {
		return "vector"
	}
	validTypes := []string{"vector", "graph", "hybrid", "keyword"}
	for _, t := range validTypes {
		if t == s {
			return s
		}
	}
	return "vector" // Default to vector if invalid
}
