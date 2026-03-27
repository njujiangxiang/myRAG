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
)

// GlobalChatHandler 处理跨知识库聊天请求
type GlobalChatHandler struct {
	sessionRepo *models.ChatSessionRepository
	messageRepo *models.MessageRepository
	kbRepo      *models.KnowledgeBaseRepository
	qdrant      *qdrant.Client
	embedding   *embedding.Client
	httpClient  *http.Client
	llmModel    string
	llmAPIKey   string
	llmBaseURL  string
}

// NewGlobalChatHandler 创建一个新的全局聊天处理器
func NewGlobalChatHandler(
	sessionRepo *models.ChatSessionRepository,
	messageRepo *models.MessageRepository,
	kbRepo *models.KnowledgeBaseRepository,
	qdrantClient *qdrant.Client,
	embeddingClient *embedding.Client,
	llmAPIKey string,
	llmModel string,
	llmProvider string,
) *GlobalChatHandler {
	llmBaseURL := "https://api.openai.com/v1"
	if llmProvider == "anthropic" {
		llmBaseURL = "https://api.anthropic.com/v1"
	}

	return &GlobalChatHandler{
		sessionRepo: sessionRepo,
		messageRepo: messageRepo,
		kbRepo:      kbRepo,
		qdrant:      qdrantClient,
		embedding:   embeddingClient,
		httpClient:  &http.Client{Timeout: 120 * time.Second},
		llmModel:    llmModel,
		llmAPIKey:   llmAPIKey,
		llmBaseURL:  llmBaseURL,
	}
}

// GlobalChatRequest 表示全局聊天请求
type GlobalChatRequest struct {
	Content string    `json:"content" binding:"required"`
	KBIDs   []uuid.UUID `json:"kb_ids" binding:"required,min=1"`
}

// GlobalChatResponse 表示全局聊天响应
type GlobalChatResponse struct {
	MessageID uuid.UUID      `json:"message_id"`
	Content   string         `json:"content"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// Chat 处理多知识库聊天
// POST /api/v1/chat
func (h *GlobalChatHandler) Chat(c *gin.Context) {
	var req GlobalChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证至少选择一个 KB
	if len(req.KBIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one knowledge base must be selected"})
		return
	}

	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user context missing"})
		return
	}

	tenantID, ok := GetTenantID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant context missing"})
		return
	}

	// 验证所有 KB 属于租户
	for _, kbID := range req.KBIDs {
		kb, err := h.kbRepo.GetByID(c.Request.Context(), kbID)
		if err != nil || kb == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("knowledge base not found: %s", kbID)})
			return
		}
		if kb.TenantID != tenantID {
			c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("access denied to knowledge base: %s", kbID)})
			return
		}
	}

	// 生成查询的嵌入向量
	queryVector, err := h.embedding.GenerateEmbedding(c.Request.Context(), req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to generate embedding: %v", err),
		})
		return
	}

	// 在所有选中的知识库中搜索
	var allResults []qdrant.SearchResult
	kbNames := make(map[string]string) // kbID -> kbName 映射
	resultKbMap := make(map[string]string) // result ID -> kbID 映射

	for _, kbID := range req.KBIDs {
		// 获取 KB 信息（名称）
		kb, err := h.kbRepo.GetByID(c.Request.Context(), kbID)
		if err == nil {
			kbNames[kbID.String()] = kb.Name
		}

		// 在这个 KB 中搜索
		results, err := h.qdrant.Search(c.Request.Context(), qdrant.SearchRequest{
			QueryVector: queryVector,
			TenantID:    tenantID,
			KBID:        kbID,
			Limit:       3, // 减少每个 KB 的 limit 以避免上下文过多
		})
		if err != nil {
			// 即使一个 KB 失败也继续
			continue
		}

		// 将结果映射到对应的 KB
		for _, result := range results {
			allResults = append(allResults, result)
			resultKbMap[result.ID.String()] = kbID.String()
		}
	}

	if len(allResults) == 0 {
		// 在任何 KB 中都没有找到结果
		c.JSON(http.StatusOK, GlobalChatResponse{
			MessageID: uuid.New(),
			Content:   "抱歉，在所选的知识库中没有找到相关信息。",
			Metadata: map[string]any{
				"sources":    []string{},
				"kb_names":   []string{},
				"kb_count":   len(req.KBIDs),
				"has_result": false,
			},
		})
		return
	}

	// 按内容去重结果（相同片段可能出现在多个 KB 中）
	seen := make(map[string]bool)
	uniqueResults := make([]qdrant.SearchResult, 0)
	for _, result := range allResults {
		if !seen[result.Content] {
			seen[result.Content] = true
			uniqueResults = append(uniqueResults, result)
		}
	}

	// 限制总结果数以避免 token 溢出
	if len(uniqueResults) > 10 {
		uniqueResults = uniqueResults[:10]
	}

	// 从搜索结果构建上下文
	var contextBuilder bytes.Buffer
	contextBuilder.WriteString("Based on the following information from multiple knowledge bases:\n\n")
	sources := make([]string, 0, len(uniqueResults))
	resultKbNames := make([]string, 0)

	for i, result := range uniqueResults {
		kbIDStr := resultKbMap[result.ID.String()]
		kbName := kbNames[kbIDStr]
		if kbName == "" {
			kbName = "Unknown KB"
		}
		contextBuilder.WriteString(fmt.Sprintf("[%d] [%s] %s\n", i+1, kbName, result.Content))
		sources = append(sources, result.DocumentID.String())
		resultKbNames = append(resultKbNames, kbName)
	}
	contextBuilder.WriteString("\n---\n\n")

	// 为 LLM 构建消息
	messages := []map[string]string{
		{"role": "system", "content": "You are a helpful assistant answering questions based on the provided context from multiple knowledge bases. Synthesize information from all sources to provide a comprehensive answer. Only use information from the context to answer. If the context doesn't contain enough information, say so."},
		{"role": "user", "content": contextBuilder.String() + req.Content},
	}

	// 使用 LLM 生成响应
	response, err := h.generateLLMResponse(c.Request.Context(), messages)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to generate response: %v", err),
		})
		return
	}

	// 获取唯一的 KB 名称
	uniqueKbNames := make(map[string]bool)
	for _, name := range resultKbNames {
		uniqueKbNames[name] = true
	}
	finalKbNames := make([]string, 0, len(uniqueKbNames))
	for name := range uniqueKbNames {
		finalKbNames = append(finalKbNames, name)
	}

	c.JSON(http.StatusOK, GlobalChatResponse{
		MessageID: uuid.New(),
		Content:   response,
		Metadata: map[string]any{
			"sources":      sources,
			"kb_names":     finalKbNames,
			"kb_count":     len(req.KBIDs),
			"doc_count":    len(sources),
			"has_result":   true,
			"context_size": len(uniqueResults),
		},
	})
}

// generateLLMResponse 使用 LLM API 生成响应
func (h *GlobalChatHandler) generateLLMResponse(ctx context.Context, messages []map[string]string) (string, error) {
	reqBody := map[string]any{
		"model":      h.llmModel,
		"messages":   messages,
		"max_tokens": 2500,
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
