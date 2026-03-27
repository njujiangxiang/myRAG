package models

import (
	"time"

	"github.com/google/uuid"
)

// Tenant represents a multi-tenant organization
type Tenant struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// User represents a system user
type User struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Never serialize to JSON
	Role         string    `json:"role"` // user, admin
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// KnowledgeBase represents a user's knowledge repository
type KnowledgeBase struct {
	ID          uuid.UUID  `json:"id"`
	TenantID    uuid.UUID  `json:"tenant_id"`
	OwnerID     uuid.UUID  `json:"owner_id"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	RAGType     string     `json:"rag_type"` // vector, graph, hybrid, keyword
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// DocumentStatus represents the processing status of a document
type DocumentStatus string

const (
	DocumentStatusPending    DocumentStatus = "pending"
	DocumentStatusProcessing DocumentStatus = "processing"
	DocumentStatusIndexed    DocumentStatus = "indexed"
	DocumentStatusError      DocumentStatus = "error"
)

// Document represents an uploaded document
type Document struct {
	ID           uuid.UUID      `json:"id"`
	TenantID     uuid.UUID      `json:"tenant_id"`
	KBID         uuid.UUID      `json:"kb_id"`
	Filename     string         `json:"filename"`
	FilePath     string         `json:"file_path"` // MinIO key
	FileSize     int64          `json:"file_size"`
	MimeType     string         `json:"mime_type"`
	Status       DocumentStatus `json:"status"`
	ErrorMessage *string        `json:"error_message,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// ChatSession represents a conversation session
type ChatSession struct {
	ID        uuid.UUID  `json:"id"`
	TenantID  uuid.UUID  `json:"tenant_id"`
	KBID      uuid.UUID  `json:"kb_id"`
	UserID    uuid.UUID  `json:"user_id"`
	Title     *string    `json:"title,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// MessageRole represents the sender of a message
type MessageRole string

const (
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
	MessageRoleSystem    MessageRole = "system"
)

// Message represents a chat message
type Message struct {
	ID        uuid.UUID   `json:"id"`
	SessionID uuid.UUID   `json:"session_id"`
	Role      MessageRole `json:"role"`
	Content   string      `json:"content"`
	Metadata  map[string]any `json:"metadata,omitempty"` // Citations, sources, etc.
	CreatedAt time.Time   `json:"created_at"`
}

// Chunk represents a processed text chunk with embedding
type Chunk struct {
	ID          uuid.UUID `json:"id"`
	DocumentID  uuid.UUID `json:"document_id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	KBID        uuid.UUID `json:"kb_id"`
	Content     string    `json:"content"`
	ChunkIndex  int       `json:"chunk_index"`
	Embedding   []float32 `json:"-"` // Vector, not stored in PostgreSQL
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}
