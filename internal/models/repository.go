package models

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// UserRepository 处理用户数据访问
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository 创建一个新的用户仓库
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create 创建一个新用户
func (r *UserRepository) Create(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (id, tenant_id, email, password_hash, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	return r.db.QueryRowContext(ctx, query,
		user.ID,
		user.TenantID,
		user.Email,
		user.PasswordHash,
		user.Role,
		time.Now(),
		time.Now(),
	).Scan(&user.ID)
}

// GetByID 根据 ID 获取用户
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	query := `
		SELECT id, tenant_id, email, password_hash, role, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	user := &User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.TenantID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetByEmail 根据邮箱获取用户
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, tenant_id, email, password_hash, role, created_at, updated_at
		FROM users
		WHERE email = $1
	`
	user := &User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.TenantID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetByTenant 获取租户的所有用户
func (r *UserRepository) GetByTenant(ctx context.Context, tenantID uuid.UUID) ([]*User, error) {
	query := `
		SELECT id, tenant_id, email, password_hash, role, created_at, updated_at
		FROM users
		WHERE tenant_id = $1
	`
	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		err := rows.Scan(
			&user.ID,
			&user.TenantID,
			&user.Email,
			&user.PasswordHash,
			&user.Role,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

// Update 更新现有用户
func (r *UserRepository) Update(ctx context.Context, user *User) error {
	query := `
		UPDATE users
		SET email = $2, password_hash = $3, role = $4, updated_at = $5
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.Role,
		time.Now(),
	)
	return err
}

// Delete 删除用户
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// TenantRepository 处理租户数据访问
type TenantRepository struct {
	db *sql.DB
}

// NewTenantRepository 创建一个新的租户仓库
func NewTenantRepository(db *sql.DB) *TenantRepository {
	return &TenantRepository{db: db}
}

// Create 创建一个新租户
func (r *TenantRepository) Create(ctx context.Context, tenant *Tenant) error {
	query := `
		INSERT INTO tenants (id, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	return r.db.QueryRowContext(ctx, query,
		tenant.ID,
		tenant.Name,
		time.Now(),
		time.Now(),
	).Scan(&tenant.ID)
}

// GetByID 根据 ID 获取租户
func (r *TenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*Tenant, error) {
	query := `
		SELECT id, name, created_at, updated_at
		FROM tenants
		WHERE id = $1
	`
	tenant := &Tenant{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&tenant.ID,
		&tenant.Name,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return tenant, nil
}

// KnowledgeBaseRepository 处理知识库数据访问
type KnowledgeBaseRepository struct {
	db *sql.DB
}

// NewKnowledgeBaseRepository 创建一个新的知识库仓库
func NewKnowledgeBaseRepository(db *sql.DB) *KnowledgeBaseRepository {
	return &KnowledgeBaseRepository{db: db}
}

// Create 创建一个新知识库
func (r *KnowledgeBaseRepository) Create(ctx context.Context, kb *KnowledgeBase) error {
	query := `
		INSERT INTO knowledge_bases (id, tenant_id, owner_id, name, description, rag_type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	return r.db.QueryRowContext(ctx, query,
		kb.ID,
		kb.TenantID,
		kb.OwnerID,
		kb.Name,
		kb.Description,
		kb.RAGType,
		time.Now(),
		time.Now(),
	).Scan(&kb.ID)
}

// GetByID 根据 ID 获取知识库
func (r *KnowledgeBaseRepository) GetByID(ctx context.Context, id uuid.UUID) (*KnowledgeBase, error) {
	query := `
		SELECT id, tenant_id, owner_id, name, description, rag_type, created_at, updated_at
		FROM knowledge_bases
		WHERE id = $1
	`
	kb := &KnowledgeBase{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&kb.ID,
		&kb.TenantID,
		&kb.OwnerID,
		&kb.Name,
		&kb.Description,
		&kb.RAGType,
		&kb.CreatedAt,
		&kb.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return kb, nil
}

// GetByTenant 获取租户的所有知识库
func (r *KnowledgeBaseRepository) GetByTenant(ctx context.Context, tenantID uuid.UUID) ([]*KnowledgeBase, error) {
	query := `
		SELECT id, tenant_id, owner_id, name, description, rag_type, created_at, updated_at
		FROM knowledge_bases
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var kbs []*KnowledgeBase
	for rows.Next() {
		kb := &KnowledgeBase{}
		err := rows.Scan(
			&kb.ID,
			&kb.TenantID,
			&kb.OwnerID,
			&kb.Name,
			&kb.Description,
			&kb.RAGType,
			&kb.CreatedAt,
			&kb.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		kbs = append(kbs, kb)
	}
	return kbs, rows.Err()
}

// Update 更新现有知识库
func (r *KnowledgeBaseRepository) Update(ctx context.Context, kb *KnowledgeBase) error {
	query := `
		UPDATE knowledge_bases
		SET name = $2, description = $3, rag_type = $4, updated_at = $5
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query,
		kb.ID,
		kb.Name,
		kb.Description,
		kb.RAGType,
		time.Now(),
	)
	return err
}

// Delete 删除知识库
func (r *KnowledgeBaseRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM knowledge_bases WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// DocumentRepository 处理文档数据访问
type DocumentRepository struct {
	db *sql.DB
}

// NewDocumentRepository 创建一个新的文档仓库
func NewDocumentRepository(db *sql.DB) *DocumentRepository {
	return &DocumentRepository{db: db}
}

// Create 创建一个新文档
func (r *DocumentRepository) Create(ctx context.Context, doc *Document) error {
	query := `
		INSERT INTO documents (id, tenant_id, kb_id, filename, file_path, file_size, mime_type, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`
	return r.db.QueryRowContext(ctx, query,
		doc.ID,
		doc.TenantID,
		doc.KBID,
		doc.Filename,
		doc.FilePath,
		doc.FileSize,
		doc.MimeType,
		doc.Status,
		time.Now(),
		time.Now(),
	).Scan(&doc.ID)
}

// GetByID 根据 ID 获取文档
func (r *DocumentRepository) GetByID(ctx context.Context, id uuid.UUID) (*Document, error) {
	query := `
		SELECT id, tenant_id, kb_id, filename, file_path, file_size, mime_type, status, error_message, metadata, created_at, updated_at
		FROM documents
		WHERE id = $1
	`
	doc := &Document{}
	var metadata []byte
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&doc.ID,
		&doc.TenantID,
		&doc.KBID,
		&doc.Filename,
		&doc.FilePath,
		&doc.FileSize,
		&doc.MimeType,
		&doc.Status,
		&doc.ErrorMessage,
		&metadata,
		&doc.CreatedAt,
		&doc.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	// 如果存在，解析 metadata JSON
	if metadata != nil {
		// TODO: 解析 JSONB metadata
	}
	return doc, nil
}

// GetByKB 获取知识库的所有文档
func (r *DocumentRepository) GetByKB(ctx context.Context, kbID uuid.UUID) ([]*Document, error) {
	query := `
		SELECT id, tenant_id, kb_id, filename, file_path, file_size, mime_type, status, error_message, created_at, updated_at
		FROM documents
		WHERE kb_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, kbID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []*Document
	for rows.Next() {
		doc := &Document{}
		err := rows.Scan(
			&doc.ID,
			&doc.TenantID,
			&doc.KBID,
			&doc.Filename,
			&doc.FilePath,
			&doc.FileSize,
			&doc.MimeType,
			&doc.Status,
			&doc.ErrorMessage,
			&doc.CreatedAt,
			&doc.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return docs, rows.Err()
}

// UpdateStatus 更新文档状态
func (r *DocumentRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status DocumentStatus, errorMsg *string) error {
	query := `
		UPDATE documents
		SET status = $2, error_message = $3, updated_at = $4
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id, status, errorMsg, time.Now())
	return err
}

// Delete 删除文档
func (r *DocumentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM documents WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// ChatSessionRepository 处理聊天会话数据访问
type ChatSessionRepository struct {
	db *sql.DB
}

// NewChatSessionRepository 创建一个新的聊天会话仓库
func NewChatSessionRepository(db *sql.DB) *ChatSessionRepository {
	return &ChatSessionRepository{db: db}
}

// Create 创建一个新聊天会话
func (r *ChatSessionRepository) Create(ctx context.Context, session *ChatSession) error {
	query := `
		INSERT INTO chat_sessions (id, tenant_id, kb_id, user_id, title, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	return r.db.QueryRowContext(ctx, query,
		session.ID,
		session.TenantID,
		session.KBID,
		session.UserID,
		session.Title,
		time.Now(),
		time.Now(),
	).Scan(&session.ID)
}

// GetByID 根据 ID 获取聊天会话
func (r *ChatSessionRepository) GetByID(ctx context.Context, id uuid.UUID) (*ChatSession, error) {
	query := `
		SELECT id, tenant_id, kb_id, user_id, title, created_at, updated_at
		FROM chat_sessions
		WHERE id = $1
	`
	session := &ChatSession{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&session.ID,
		&session.TenantID,
		&session.KBID,
		&session.UserID,
		&session.Title,
		&session.CreatedAt,
		&session.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return session, nil
}

// GetByKB 获取知识库的所有会话
func (r *ChatSessionRepository) GetByKB(ctx context.Context, kbID uuid.UUID) ([]*ChatSession, error) {
	query := `
		SELECT id, tenant_id, kb_id, user_id, title, created_at, updated_at
		FROM chat_sessions
		WHERE kb_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, kbID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*ChatSession
	for rows.Next() {
		session := &ChatSession{}
		err := rows.Scan(
			&session.ID,
			&session.TenantID,
			&session.KBID,
			&session.UserID,
			&session.Title,
			&session.CreatedAt,
			&session.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}
	return sessions, rows.Err()
}

// Delete 删除聊天会话
func (r *ChatSessionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM chat_sessions WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// MessageRepository 处理消息数据访问
type MessageRepository struct {
	db *sql.DB
}

// NewMessageRepository 创建一个新的消息仓库
func NewMessageRepository(db *sql.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

// Create 创建一个新消息
func (r *MessageRepository) Create(ctx context.Context, msg *Message) error {
	query := `
		INSERT INTO messages (id, session_id, role, content, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	return r.db.QueryRowContext(ctx, query,
		msg.ID,
		msg.SessionID,
		msg.Role,
		msg.Content,
		time.Now(),
	).Scan(&msg.ID)
}

// GetBySession 获取会话的所有消息
func (r *MessageRepository) GetBySession(ctx context.Context, sessionID uuid.UUID) ([]*Message, error) {
	query := `
		SELECT id, session_id, role, content, created_at
		FROM messages
		WHERE session_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		msg := &Message{}
		err := rows.Scan(
			&msg.ID,
			&msg.SessionID,
			&msg.Role,
			&msg.Content,
			&msg.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

// DeleteBySession 删除会话的所有消息
func (r *MessageRepository) DeleteBySession(ctx context.Context, sessionID uuid.UUID) error {
	query := `DELETE FROM messages WHERE session_id = $1`
	_, err := r.db.ExecContext(ctx, query, sessionID)
	return err
}
