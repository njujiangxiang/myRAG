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

// DocumentHandler handles document requests
type DocumentHandler struct {
	docRepo  *models.DocumentRepository
	kbRepo   *models.KnowledgeBaseRepository
	minio    *minio.Client
	nats     *nats.Client
	qdrant   *qdrant.Client
}

// NewDocumentHandler creates a new document handler
func NewDocumentHandler(docRepo *models.DocumentRepository, kbRepo *models.KnowledgeBaseRepository, minioClient *minio.Client, natsClient *nats.Client, qdrantClient *qdrant.Client) *DocumentHandler {
	return &DocumentHandler{
		docRepo:  docRepo,
		kbRepo:   kbRepo,
		minio:    minioClient,
		nats:     natsClient,
		qdrant:   qdrantClient,
	}
}

// DocumentResult represents document data in response
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

// ListDocuments handles listing documents for a knowledge base
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

// UploadDocumentRequest represents a document upload response
type UploadDocumentResponse struct {
	ID       uuid.UUID `json:"id"`
	Filename string    `json:"filename"`
	Status   string    `json:"status"`
	Message  string    `json:"message"`
}

// UploadDocument handles uploading a new document
// POST /api/v1/kbs/:id/docs
func (h *DocumentHandler) UploadDocument(c *gin.Context) {
	kbIDStr := c.Param("id")
	kbID, err := uuid.Parse(kbIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid knowledge base ID"})
		return
	}

	// Get tenant from context
	tenantID, ok := GetTenantID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant context missing"})
		return
	}

	// Verify KB belongs to tenant
	kb, err := h.kbRepo.GetByID(c.Request.Context(), kbID)
	if err != nil || kb == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "knowledge base not found"})
		return
	}
	if kb.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Get file from form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing file"})
		return
	}
	defer file.Close()

	// Validate file size (max 50MB)
	const maxFileSize = 50 << 20 // 50MB
	if header.Size > maxFileSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file too large (max 50MB)"})
		return
	}

	// Sanitize filename (prevent path traversal and special characters)
	safeFilename := path.Base(header.Filename)
	safeFilename = strings.Map(func(r rune) rune {
		// Allow alphanumeric, dots, underscores, hyphens
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			return r
		}
		return '_'
	}, safeFilename)

	// Validate file type
	allowedMimeTypes := map[string]bool{
		"application/pdf": true,
		"text/csv":        true,
		"text/markdown":   true,
		"text/plain":      true,
	}

	contentType := header.Header.Get("Content-Type")

	// If Content-Type is empty or unknown, try to detect it from file extension
	if !allowedMimeTypes[contentType] {
		// Check if it's an empty/unknown content type (common with multipart uploads)
		if contentType == "" || contentType == "application/octet-stream" {
			// Detect MIME type from file extension
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
			}
		}

		// Final validation
		if !allowedMimeTypes[contentType] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported file type"})
			return
		}
	}

	// Read file content
	fileData, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}

	// Generate unique object key: documents/{tenant_id}/{kb_id}/{doc_id}_{filename}
	docID := uuid.New()
	objectKey := fmt.Sprintf("documents/%s/%s/%s_%s",
		tenantID.String(),
		kbID.String(),
		docID.String(),
		safeFilename,
	)

	// Upload to MinIO
	uploadedPath, err := h.minio.UploadFile(c.Request.Context(), objectKey, fileData, contentType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload file"})
		return
	}

	// Create document record
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
		// Clean up MinIO file on DB error
		_ = h.minio.DeleteFile(c.Request.Context(), objectKey)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create document record"})
		return
	}

	// Publish event to NATS for async processing
	event := worker.DocumentEvent{
		DocID:    doc.ID,
		TenantID: doc.TenantID,
		KBID:     doc.KBID,
		FilePath: uploadedPath,
		MimeType: doc.MimeType,
	}
	eventData, _ := json.Marshal(event)
	if err := h.nats.PublishDocumentEvent(c.Request.Context(), "uploaded", doc.ID.String(), eventData); err != nil {
		// Log error but don't fail the request (document is still uploaded)
		// Worker will process it eventually or it will timeout
		fmt.Printf("failed to publish NATS event: %v\n", err)
	}

	c.JSON(http.StatusAccepted, UploadDocumentResponse{
		ID:       doc.ID,
		Filename: doc.Filename,
		Status:   string(doc.Status),
		Message:  "file uploaded, processing in background",
	})
}

// GetDocument handles getting a single document
// GET /api/v1/docs/:id
func (h *DocumentHandler) GetDocument(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document ID"})
		return
	}

	// Get tenant ID from context
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

	// Verify tenant ownership
	if doc.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Verify KB belongs to tenant
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

// GetDocumentContent handles getting parsed document content
// GET /api/v1/docs/:id/content
func (h *DocumentHandler) GetDocumentContent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document ID"})
		return
	}

	// Get tenant ID from context
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

	// Verify tenant ownership
	if doc.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Verify KB belongs to tenant
	kb, err := h.kbRepo.GetByID(c.Request.Context(), doc.KBID)
	if err != nil || kb == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "knowledge base not found"})
		return
	}
	if kb.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Check if document is indexed
	if doc.Status != models.DocumentStatusIndexed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "document not yet indexed"})
		return
	}

	// Generate presigned URL for temporary access (15 minutes expiry)
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

// DeleteDocument handles deleting a document
// DELETE /api/v1/docs/:id
func (h *DocumentHandler) DeleteDocument(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document ID"})
		return
	}

	// Get tenant ID from context
	tenantID, ok := GetTenantID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant context missing"})
		return
	}

	// Get document to find MinIO key
	doc, err := h.docRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
		return
	}

	// Verify tenant ownership
	if doc.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Verify KB belongs to tenant
	kb, err := h.kbRepo.GetByID(c.Request.Context(), doc.KBID)
	if err != nil || kb == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "knowledge base not found"})
		return
	}
	if kb.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	// Delete from MinIO first
	if doc.FilePath != "" {
		objectKey := path.Base(doc.FilePath)
		_ = h.minio.DeleteFile(c.Request.Context(), objectKey)
		// Continue even if delete fails (file might not exist)
	}

	// Delete vectors from Qdrant
	if err := h.qdrant.DeleteByDocumentID(c.Request.Context(), doc.TenantID, doc.KBID, doc.ID); err != nil {
		// Log error but continue (document will still be deleted from DB)
		fmt.Printf("failed to delete qdrant vectors: %v\n", err)
	}

	// Delete from database
	if err := h.docRepo.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete document"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "document deleted"})
}
