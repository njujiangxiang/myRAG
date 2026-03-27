package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"myrag/internal/models"
)

// KnowledgeBaseHandler 处理知识库请求
type KnowledgeBaseHandler struct {
	kbRepo *models.KnowledgeBaseRepository
}

// NewKnowledgeBaseHandler 创建一个新的 KB 处理器
func NewKnowledgeBaseHandler(kbRepo *models.KnowledgeBaseRepository) *KnowledgeBaseHandler {
	return &KnowledgeBaseHandler{
		kbRepo: kbRepo,
	}
}

// CreateKBRequest 表示创建知识库请求
type CreateKBRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description,omitempty"`
	RAGType     string  `json:"rag_type,omitempty"` // vector, graph, hybrid, keyword
}

// KBResult 表示响应中的知识库数据
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

// ListKBs 列出当前租户的知识库
// GET /api/v1/kbs
func (h *KnowledgeBaseHandler) ListKBs(c *gin.Context) {
	tenantID, ok := GetTenantID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant context missing"})
		return
	}

	kbs, err := h.kbRepo.GetByTenant(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取知识库列表失败"})
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

// GetKB 获取单个知识库
// GET /api/v1/kbs/:id
func (h *KnowledgeBaseHandler) GetKB(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的知识库 ID"})
		return
	}

	// 从上下文中获取租户 ID
	tenantID, ok := GetTenantID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant context missing"})
		return
	}

	kb, err := h.kbRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "知识库不存在"})
		return
	}

	// 验证租户所有权
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

// CreateKB 创建新知识库
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建知识库失败"})
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

// UpdateKB 更新知识库
// PUT /api/v1/kbs/:id
func (h *KnowledgeBaseHandler) UpdateKB(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的知识库 ID"})
		return
	}

	// 从上下文中获取租户 ID
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
		c.JSON(http.StatusNotFound, gin.H{"error": "知识库不存在"})
		return
	}

	// 验证租户所有权
	if kb.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	kb.Name = req.Name
	kb.Description = req.Description
	kb.RAGType = getRAGType(req.RAGType)
	kb.UpdatedAt = time.Now()

	if err := h.kbRepo.Update(c.Request.Context(), kb); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新知识库失败"})
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

// DeleteKB 删除知识库
// DELETE /api/v1/kbs/:id
func (h *KnowledgeBaseHandler) DeleteKB(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的知识库 ID"})
		return
	}

	// 从上下文中获取租户 ID
	tenantID, ok := GetTenantID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant context missing"})
		return
	}

	kb, err := h.kbRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "知识库不存在"})
		return
	}

	// 验证租户所有权
	if kb.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	if err := h.kbRepo.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除知识库失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "知识库已删除"})
}

// kbResponse 是格式化知识库响应的辅助函数
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

// getRAGType 返回有效的 RAG 类型，否则返回默认值
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
	return "vector" // 如果无效，默认为 vector
}
