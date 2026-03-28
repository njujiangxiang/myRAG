package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"myrag/internal/embedding"
	"myrag/internal/models"
	"myrag/internal/qdrant"
	"myrag/internal/rag"
)

// ChatHandler 处理聊天请求
type ChatHandler struct {
	sessionRepo *models.ChatSessionRepository
	messageRepo *models.MessageRepository
	kbRepo      *models.KnowledgeBaseRepository
	qdrant      *qdrant.Client
	embedding   *embedding.Client
	ragFactory  *rag.Factory
	httpClient  *http.Client
	llmModel    string
	llmAPIKey   string
	llmBaseURL  string
}

// NewChatHandler 创建一个新的聊天处理器
func NewChatHandler(
	sessionRepo *models.ChatSessionRepository,
	messageRepo *models.MessageRepository,
	kbRepo *models.KnowledgeBaseRepository,
	qdrantClient *qdrant.Client,
	embeddingClient *embedding.Client,
	ragFactory *rag.Factory,
	llmAPIKey string,
	llmModel string,
	llmProvider string,
	llmBaseURL string,
) *ChatHandler {
	if llmBaseURL == "" {
		llmBaseURL = "https://api.openai.com/v1"
		if llmProvider == "anthropic" {
			llmBaseURL = "https://api.anthropic.com/v1"
		}
	}

	return &ChatHandler{
		sessionRepo: sessionRepo,
		messageRepo: messageRepo,
		kbRepo:      kbRepo,
		qdrant:      qdrantClient,
		embedding:   embeddingClient,
		ragFactory:  ragFactory,
		httpClient:  &http.Client{Timeout: 60 * time.Second},
		llmModel:    llmModel,
		llmAPIKey:   llmAPIKey,
		llmBaseURL:  llmBaseURL,
	}
}

// CreateChatRequest 表示聊天请求
type CreateChatRequest struct {
	Title   *string `json:"title,omitempty"`
	Content string  `json:"content" binding:"required"`
}

// ChatMessage 表示响应中的聊天消息
type ChatMessage struct {
	ID        uuid.UUID      `json:"id"`
	Role      string         `json:"role"`
	Content   string         `json:"content"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

// ChatSessionResult 表示响应中的聊天会话
type ChatSessionResult struct {
	ID        uuid.UUID  `json:"id"`
	KBID      uuid.UUID  `json:"kb_id"`
	UserID    uuid.UUID  `json:"user_id"`
	Title     *string    `json:"title,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// ChatResponse 表示聊天响应（支持 SSE 流式传输）
type ChatResponse struct {
	MessageID uuid.UUID      `json:"message_id"`
	Content   string         `json:"content"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// Chat 处理知识库聊天
// POST /api/v1/kbs/:id/chat
func (h *ChatHandler) Chat(c *gin.Context) {
	kbIDStr := c.Param("id")
	kbID, err := uuid.Parse(kbIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid knowledge base ID"})
		return
	}

	var req CreateChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 从上下文中获取用户和租户
	userID, ok := GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user context missing"})
		return
	}

	tenantID, exists := GetTenantID(c)
	if !exists {
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

	session := &models.ChatSession{
		ID:        uuid.New(),
		TenantID:  tenantID,
		KBID:      kbID,
		UserID:    userID,
		Title:     req.Title,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.sessionRepo.Create(c.Request.Context(), session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create chat session"})
		return
	}

	// 保存用户消息
	userMsg := &models.Message{
		ID:        uuid.New(),
		SessionID: session.ID,
		Role:      models.MessageRoleUser,
		Content:   req.Content,
		CreatedAt: time.Now(),
	}

	if err := h.messageRepo.Create(c.Request.Context(), userMsg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save user message"})
		return
	}

	// 根据 KB 类型获取 RAG 策略
	strategy, err := h.ragFactory.GetStrategyByKB(c.Request.Context(), kbID, tenantID, h.kbRepo)
	if err != nil {
		errorMsg := fmt.Sprintf("failed to get RAG strategy: %v", err)
		_ = h.messageRepo.Create(c.Request.Context(), &models.Message{
			ID:        uuid.New(),
			SessionID: session.ID,
			Role:      models.MessageRoleAssistant,
			Content:   "Sorry, I encountered an error with the RAG configuration.",
			CreatedAt: time.Now(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorMsg})
		return
	}

	// RAG：使用策略搜索相关片段
	searchResults, err := strategy.Search(c.Request.Context(), req.Content, kbID, tenantID, 5)
	if err != nil {
		errorMsg := fmt.Sprintf("search failed: %v", err)
		_ = h.messageRepo.Create(c.Request.Context(), &models.Message{
			ID:        uuid.New(),
			SessionID: session.ID,
			Role:      models.MessageRoleAssistant,
			Content:   "Sorry, I encountered an error searching the knowledge base.",
			CreatedAt: time.Now(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": errorMsg})
		return
	}

	// 从搜索结果构建上下文
	var contextBuilder bytes.Buffer
	contextBuilder.WriteString("Based on the following information from the knowledge base:\n\n")
	sources := make([]string, 0, len(searchResults))
	for i, result := range searchResults {
		contextBuilder.WriteString(fmt.Sprintf("[%d] %s\n", i+1, result.Content))
		sources = append(sources, result.DocumentID.String())
	}
	contextBuilder.WriteString("\n---\n\n")

	// 为 LLM 构建消息
	messages := []map[string]string{
		{"role": "system", "content": "You are a helpful assistant answering questions based on the provided context. Only use information from the context to answer. If the context doesn't contain enough information, say so."},
		{"role": "user", "content": contextBuilder.String() + req.Content},
	}

	// 使用 LLM 生成响应
	response, err := h.generateLLMResponse(c.Request.Context(), messages)
	if err != nil {
		_ = h.messageRepo.Create(c.Request.Context(), &models.Message{
			ID:        uuid.New(),
			SessionID: session.ID,
			Role:      models.MessageRoleAssistant,
			Content:   "Sorry, I encountered an error generating the response.",
			CreatedAt: time.Now(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate response"})
		return
	}

	// 保存助手消息
	assistantMsg := &models.Message{
		ID:        uuid.New(),
		SessionID: session.ID,
		Role:      models.MessageRoleAssistant,
		Content:   response,
		Metadata:  map[string]any{"sources": sources, "context_chunks": len(searchResults)},
		CreatedAt: time.Now(),
	}

	if err := h.messageRepo.Create(c.Request.Context(), assistantMsg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save assistant message"})
		return
	}

	c.JSON(http.StatusOK, ChatResponse{
		MessageID: assistantMsg.ID,
		Content:   assistantMsg.Content,
		Metadata:  assistantMsg.Metadata,
	})
}

// generateLLMResponse 使用 LLM API 生成响应
func (h *ChatHandler) generateLLMResponse(ctx context.Context, messages []map[string]string) (string, error) {
	reqBody := map[string]any{
		"model":    h.llmModel,
		"messages": messages,
		"max_tokens": 2000,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", h.llmBaseURL+"/chat/completions", bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.llmAPIKey)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return result.Choices[0].Message.Content, nil
}

// ChatStream 处理 SSE 流式聊天
// GET /api/v1/kbs/:id/chat/stream
func (h *ChatHandler) ChatStream(c *gin.Context) {
	// 设置 SSE 头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// TODO: Implement SSE streaming
	// For now, send a placeholder event
	c.SSEvent("message", gin.H{
		"content": "SSE streaming coming soon",
	})
}

// ListSessions 处理列出知识库的聊天会话
// GET /api/v1/kbs/:id/sessions
func (h *ChatHandler) ListSessions(c *gin.Context) {
	kbIDStr := c.Param("id")
	kbID, err := uuid.Parse(kbIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid knowledge base ID"})
		return
	}

	sessions, err := h.sessionRepo.GetByKB(c.Request.Context(), kbID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list chat sessions"})
		return
	}

	results := make([]ChatSessionResult, len(sessions))
	for i, session := range sessions {
		results[i] = ChatSessionResult{
			ID:        session.ID,
			KBID:      session.KBID,
			UserID:    session.UserID,
			Title:     session.Title,
			CreatedAt: session.CreatedAt,
			UpdatedAt: session.UpdatedAt,
		}
	}

	c.JSON(http.StatusOK, results)
}

// GetSessionMessages 处理获取会话消息
// GET /api/v1/sessions/:id/messages
func (h *ChatHandler) GetSessionMessages(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	messages, err := h.messageRepo.GetBySession(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get messages"})
		return
	}

	results := make([]ChatMessage, len(messages))
	for i, msg := range messages {
		results[i] = ChatMessage{
			ID:        msg.ID,
			Role:      string(msg.Role),
			Content:   msg.Content,
			Metadata:  msg.Metadata,
			CreatedAt: msg.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, results)
}

// DeleteSession 处理删除聊天会话
// DELETE /api/v1/sessions/:id
func (h *ChatHandler) DeleteSession(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session ID"})
		return
	}

	// 先删除消息（级联或手动）
	if err := h.messageRepo.DeleteBySession(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete messages"})
		return
	}

	if err := h.sessionRepo.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "session deleted"})
}
