package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"myrag/internal/minio"
	"myrag/internal/models"
	"myrag/internal/nats"
	"myrag/internal/qdrant"
	"myrag/internal/worker"
)

// DocumentHandler 处理文档请求
type DocumentHandler struct {
	docRepo  *models.DocumentRepository
	kbRepo   *models.KnowledgeBaseRepository
	minio    *minio.Client
	nats     *nats.Client
	qdrant   *qdrant.Client
}

// NewDocumentHandler 创建一个新的文档处理器
func NewDocumentHandler(docRepo *models.DocumentRepository, kbRepo *models.KnowledgeBaseRepository, minioClient *minio.Client, natsClient *nats.Client, qdrantClient *qdrant.Client) *DocumentHandler {
	return &DocumentHandler{
		docRepo:  docRepo,
		kbRepo:   kbRepo,
		minio:    minioClient,
		nats:     natsClient,
		qdrant:   qdrantClient,
	}
}

// DocumentResult 表示响应中的文档数据
type DocumentResult struct {
	ID           uuid.UUID      `json:"id"`
	TenantID     uuid.UUID      `json:"tenant_id"`
	KBID         uuid.UUID      `json:"kb_id"`
	Filename     string         `json:"filename"`
	FileSize     int64          `json:"file_size"`
	MimeType     string         `json:"mime_type"`
	Status       string         `json:"status"`
	ErrorMessage *string        `json:"error_message,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// ListDocuments 处理列出知识库文档的请求
// GET /api/v1/kbs/:id/docs
func (h *DocumentHandler) ListDocuments(c *gin.Context) {
	kbIDStr := c.Param("id")
	kbID, err := uuid.Parse(kbIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid knowledge base ID"})
		return
	}

	docs, err := h.docRepo.GetByKB(c.Request.Context(), kbID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list documents"})
		return
	}

	results := make([]DocumentResult, len(docs))
	for i, doc := range docs {
		results[i] = DocumentResult{
			ID:           doc.ID,
			TenantID:     doc.TenantID,
			KBID:         doc.KBID,
			Filename:     doc.Filename,
			FileSize:     doc.FileSize,
			MimeType:     doc.MimeType,
			Status:       string(doc.Status),
			ErrorMessage: doc.ErrorMessage,
			CreatedAt:    doc.CreatedAt,
			UpdatedAt:    doc.UpdatedAt,
		}
	}

	c.JSON(http.StatusOK, results)
}

// UploadDocumentResponse 表示文档上传响应
type UploadDocumentResponse struct {
	ID       uuid.UUID `json:"id"`
	Filename string    `json:"filename"`
	Status   string    `json:"status"`
	Message  string    `json:"message"`
}

// UploadDocument 处理上传新文档
// POST /api/v1/kbs/:id/docs
func (h *DocumentHandler) UploadDocument(c *gin.Context) {
	kbIDStr := c.Param("id")
	kbID, err := uuid.Parse(kbIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid knowledge base ID"})
		return
	}

	// 从上下文中获取租户
	tenantID, ok := GetTenantID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant context missing"})
		return
	}

	// 验证 KB 属于租户
	kb, err := h.kbRepo.GetByID(c.Request.Context(), kbID)
	if err != nil || kb == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "knowledge base not found"})
		return
	}
	if kb.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// 从表单中获取文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing file"})
		return
	}
	defer file.Close()

	// 验证文件大小（最大 50MB）
	const maxFileSize = 50 << 20 // 50MB
	if header.Size > maxFileSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file too large (max 50MB)"})
		return
	}

	// 清理文件名（防止路径遍历和特殊字符）
	safeFilename := path.Base(header.Filename)
	safeFilename = strings.Map(func(r rune) rune {
		// 允许字母数字、点、下划线、连字符
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			return r
		}
		return '_'
	}, safeFilename)

	// 验证文件类型
	allowedMimeTypes := map[string]bool{
		"application/pdf":      true,
		"text/csv":             true,
		"text/markdown":        true,
		"text/plain":           true,
		"application/msword":               true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	}

	contentType := header.Header.Get("Content-Type")

	// 如果 Content-Type 为空或未知，尝试从文件扩展名检测
	if !allowedMimeTypes[contentType] {
		// 检查是否是空/未知内容类型（常见于 multipart 上传）
		if contentType == "" || contentType == "application/octet-stream" {
			// 从文件扩展名检测 MIME 类型
			ext := strings.ToLower(path.Ext(safeFilename))
			switch ext {
			case ".pdf":
				contentType = "application/pdf"
			case ".md", ".markdown":
				contentType = "text/markdown"
			case ".txt":
				contentType = "text/plain"
			case ".csv":
				contentType = "text/csv"
			case ".doc":
				contentType = "application/msword"
			case ".docx":
				contentType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
			}
		}

		// 最终验证
		if !allowedMimeTypes[contentType] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported file type"})
			return
		}
	}

	// 读取文件内容
	fileData, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}

	// 生成唯一的对象键：documents/{tenant_id}/{kb_id}/{doc_id}_{filename}
	docID := uuid.New()
	objectKey := fmt.Sprintf("documents/%s/%s/%s_%s",
		tenantID.String(),
		kbID.String(),
		docID.String(),
		safeFilename,
	)

	// 上传到 MinIO
	uploadedPath, err := h.minio.UploadFile(c.Request.Context(), objectKey, fileData, contentType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload file"})
		return
	}

	// 创建文档记录
	doc := &models.Document{
		ID:        docID,
		TenantID:  tenantID,
		KBID:      kbID,
		Filename:  header.Filename,
		FilePath:  uploadedPath,
		FileSize:  header.Size,
		MimeType:  contentType,
		Status:    models.DocumentStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.docRepo.Create(c.Request.Context(), doc); err != nil {
		// 如果 DB 错误，清理 MinIO 文件
		_ = h.minio.DeleteFile(c.Request.Context(), objectKey)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create document record"})
		return
	}

	// 发布事件到 NATS 进行异步处理
	event := worker.DocumentEvent{
		DocID:    doc.ID,
		TenantID: doc.TenantID,
		KBID:     doc.KBID,
		FilePath: uploadedPath,
		MimeType: doc.MimeType,
	}
	eventData, _ := json.Marshal(event)
	if err := h.nats.PublishDocumentEvent(c.Request.Context(), "uploaded", doc.ID.String(), eventData); err != nil {
		// 记录错误但不失败（文档仍已上传）
		// Worker 将最终处理它，否则会超时
		fmt.Printf("failed to publish NATS event: %v\n", err)
	}

	c.JSON(http.StatusAccepted, UploadDocumentResponse{
		ID:       doc.ID,
		Filename: doc.Filename,
		Status:   string(doc.Status),
		Message:  "file uploaded, processing in background",
	})
}

// GetDocument 处理获取单个文档
// GET /api/v1/docs/:id
func (h *DocumentHandler) GetDocument(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document ID"})
		return
	}

	// 从上下文中获取租户 ID
	tenantID, ok := GetTenantID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant context missing"})
		return
	}

	doc, err := h.docRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return
	}

	// 验证租户所有权
	if doc.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// 验证 KB 属于租户
	kb, err := h.kbRepo.GetByID(c.Request.Context(), doc.KBID)
	if err != nil || kb == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "knowledge base not found"})
		return
	}
	if kb.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	c.JSON(http.StatusOK, DocumentResult{
		ID:           doc.ID,
		TenantID:     doc.TenantID,
		KBID:         doc.KBID,
		Filename:     doc.Filename,
		FileSize:     doc.FileSize,
		MimeType:     doc.MimeType,
		Status:       string(doc.Status),
		ErrorMessage: doc.ErrorMessage,
		CreatedAt:    doc.CreatedAt,
		UpdatedAt:    doc.UpdatedAt,
	})
}

// GetDocumentContent 处理获取解析后的文档内容
// GET /api/v1/docs/:id/content
func (h *DocumentHandler) GetDocumentContent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document ID"})
		return
	}

	// 从上下文中获取租户 ID
	tenantID, ok := GetTenantID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant context missing"})
		return
	}

	doc, err := h.docRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return
	}

	// 验证租户所有权
	if doc.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// 验证 KB 属于租户
	kb, err := h.kbRepo.GetByID(c.Request.Context(), doc.KBID)
	if err != nil || kb == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "knowledge base not found"})
		return
	}
	if kb.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// 检查文档是否已索引
	if doc.Status != models.DocumentStatusIndexed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "document not yet indexed"})
		return
	}

	// 生成预签名 URL 以供临时访问（15 分钟过期）
	presignedURL, err := h.minio.GetPresignedURL(c.Request.Context(), path.Base(doc.FilePath), 15*time.Minute)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate download URL"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"download_url": presignedURL,
		"filename":     doc.Filename,
		"mime_type":    doc.MimeType,
		"file_size":    doc.FileSize,
	})
}

// DeleteDocument 处理删除文档
// DELETE /api/v1/docs/:id
func (h *DocumentHandler) DeleteDocument(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document ID"})
		return
	}

	// 从上下文中获取租户 ID
	tenantID, ok := GetTenantID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant context missing"})
		return
	}

	// 获取文档以查找 MinIO 键
	doc, err := h.docRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return
	}

	// 验证租户所有权
	if doc.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// 验证 KB 属于租户
	kb, err := h.kbRepo.GetByID(c.Request.Context(), doc.KBID)
	if err != nil || kb == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "knowledge base not found"})
		return
	}
	if kb.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// 先从 MinIO 删除
	if doc.FilePath != "" {
		objectKey := path.Base(doc.FilePath)
		_ = h.minio.DeleteFile(c.Request.Context(), objectKey)
		// 即使删除失败也继续（文件可能不存在）
	}

	// 从 Qdrant 删除向量
	if err := h.qdrant.DeleteByDocumentID(c.Request.Context(), doc.TenantID, doc.KBID, doc.ID); err != nil {
		// 记录错误但继续（文档仍将从 DB 删除）
		fmt.Printf("failed to delete qdrant vectors: %v\n", err)
	}

	// 从数据库删除
	if err := h.docRepo.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete document"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "document deleted"})
}
