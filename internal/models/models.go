package models

import (
	"time"

	"github.com/google/uuid"
)

// Tenant 表示多租户组织
type Tenant struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// User 表示系统用户
type User struct {
	ID           uuid.UUID `json:"id"`
	TenantID     uuid.UUID `json:"tenant_id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // 永不序列化为 JSON
	Role         string    `json:"role"` // user, admin
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// KnowledgeBase 表示用户的知识库
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

// DocumentStatus 表示文档的处理状态
type DocumentStatus string

const (
	DocumentStatusPending    DocumentStatus = "pending"    // 待处理
	DocumentStatusProcessing DocumentStatus = "processing" // 处理中
	DocumentStatusIndexed    DocumentStatus = "indexed"    // 已索引
	DocumentStatusError      DocumentStatus = "error"      // 错误
)

// Document 表示上传的文档
type Document struct {
	ID           uuid.UUID      `json:"id"`
	TenantID     uuid.UUID      `json:"tenant_id"`
	KBID         uuid.UUID      `json:"kb_id"`
	Filename     string         `json:"filename"`
	FilePath     string         `json:"file_path"` // MinIO 对象键
	FileSize     int64          `json:"file_size"`
	MimeType     string         `json:"mime_type"`
	Status       DocumentStatus `json:"status"`
	ErrorMessage *string        `json:"error_message,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// ChatSession 表示对话会话
type ChatSession struct {
	ID        uuid.UUID  `json:"id"`
	TenantID  uuid.UUID  `json:"tenant_id"`
	KBID      uuid.UUID  `json:"kb_id"`
	UserID    uuid.UUID  `json:"user_id"`
	Title     *string    `json:"title,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// MessageRole 表示消息发送者
type MessageRole string

const (
	MessageRoleUser      MessageRole = "user"      // 用户
	MessageRoleAssistant MessageRole = "assistant" // 助手
	MessageRoleSystem    MessageRole = "system"    // 系统
)

// Message 表示聊天消息
type Message struct {
	ID        uuid.UUID   `json:"id"`
	SessionID uuid.UUID   `json:"session_id"`
	Role      MessageRole `json:"role"`
	Content   string      `json:"content"`
	Metadata  map[string]any `json:"metadata,omitempty"` // 引用、来源等
	CreatedAt time.Time   `json:"created_at"`
}

// Chunk 表示带有嵌入向量的处理文本块
type Chunk struct {
	ID          uuid.UUID `json:"id"`
	DocumentID  uuid.UUID `json:"document_id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	KBID        uuid.UUID `json:"kb_id"`
	Content     string    `json:"content"`
	ChunkIndex  int       `json:"chunk_index"`
	Embedding   []float32 `json:"-"` // 向量，不存储在 PostgreSQL 中
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}
