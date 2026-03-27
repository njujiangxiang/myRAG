package models

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// UserRepository handles user data access
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user
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

// GetByID retrieves a user by ID
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

// GetByEmail retrieves a user by email
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

// GetByTenant retrieves all users for a tenant
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

// Update updates an existing user
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

// Delete deletes a user
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// TenantRepository handles tenant data access
type TenantRepository struct {
	db *sql.DB
}

// NewTenantRepository creates a new tenant repository
func NewTenantRepository(db *sql.DB) *TenantRepository {
	return &TenantRepository{db: db}
}

// Create creates a new tenant
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

// GetByID retrieves a tenant by ID
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

// KnowledgeBaseRepository handles knowledge base data access
type KnowledgeBaseRepository struct {
	db *sql.DB
}

// NewKnowledgeBaseRepository creates a new KB repository
func NewKnowledgeBaseRepository(db *sql.DB) *KnowledgeBaseRepository {
	return &KnowledgeBaseRepository{db: db}
}

// Create creates a new knowledge base
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

// GetByID retrieves a knowledge base by ID
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

// GetByTenant retrieves all knowledge bases for a tenant
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

// Update updates an existing knowledge base
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

// Delete deletes a knowledge base
func (r *KnowledgeBaseRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM knowledge_bases WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// DocumentRepository handles document data access
type DocumentRepository struct {
	db *sql.DB
}

// NewDocumentRepository creates a new document repository
func NewDocumentRepository(db *sql.DB) *DocumentRepository {
	return &DocumentRepository{db: db}
}

// Create creates a new document
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

// GetByID retrieves a document by ID
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
	// Parse metadata JSON if present
	if metadata != nil {
		// TODO: Parse JSONB metadata
	}
	return doc, nil
}

// GetByKB retrieves all documents for a knowledge base
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

// UpdateStatus updates document status
func (r *DocumentRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status DocumentStatus, errorMsg *string) error {
	query := `
		UPDATE documents
		SET status = $2, error_message = $3, updated_at = $4
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id, status, errorMsg, time.Now())
	return err
}

// Delete deletes a document
func (r *DocumentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM documents WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// ChatSessionRepository handles chat session data access
type ChatSessionRepository struct {
	db *sql.DB
}

// NewChatSessionRepository creates a new chat session repository
func NewChatSessionRepository(db *sql.DB) *ChatSessionRepository {
	return &ChatSessionRepository{db: db}
}

// Create creates a new chat session
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

// GetByID retrieves a chat session by ID
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

// GetByKB retrieves all sessions for a knowledge base
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

// Delete deletes a chat session
func (r *ChatSessionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM chat_sessions WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// MessageRepository handles message data access
type MessageRepository struct {
	db *sql.DB
}

// NewMessageRepository creates a new message repository
func NewMessageRepository(db *sql.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

// Create creates a new message
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

// GetBySession retrieves all messages for a session
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

// DeleteBySession deletes all messages for a session
func (r *MessageRepository) DeleteBySession(ctx context.Context, sessionID uuid.UUID) error {
	query := `DELETE FROM messages WHERE session_id = $1`
	_, err := r.db.ExecContext(ctx, query, sessionID)
	return err
}
