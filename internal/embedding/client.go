package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// Client 处理嵌入生成
type Client struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// Config 持有嵌入客户端配置
type Config struct {
	APIKey  string
	BaseURL string
	Model   string
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = "sk-placeholder"
	}

	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	model := os.Getenv("EMBEDDING_MODEL")
	if model == "" {
		model = "text-embedding-3-small"
	}

	return Config{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
	}
}

// New 创建一个新的嵌入客户端
func New(config Config) *Client {
	return &Client{
		apiKey:  config.APIKey,
		baseURL: config.BaseURL,
		model:   config.Model,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute, // 增加超时时间以支持本地慢速模型
		},
	}
}

// EmbeddingRequest 表示 OpenAI 嵌入请求
type EmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// EmbeddingResponse 表示 OpenAI 嵌入响应
type EmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// GenerateEmbeddings 批量生成文本的嵌入向量
func (c *Client) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	// 减小批量大小以避免请求过大导致超时
	// 对于本地 Ollama 部署，使用较小的批量
	batchSize := 10
	var allEmbeddings [][]float32

	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}

		batch := texts[i:end]
		embeddings, err := c.generateBatch(ctx, batch)
		if err != nil {
			return nil, fmt.Errorf("生成批量嵌入失败，起始索引 %d: %w", i, err)
		}

		allEmbeddings = append(allEmbeddings, embeddings...)
	}

	return allEmbeddings, nil
}

// generateBatch 为单批文本生成嵌入向量
func (c *Client) generateBatch(ctx context.Context, texts []string) ([][]float32, error) {
	reqBody := EmbeddingRequest{
		Model: c.model,
		Input: texts,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败：%w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/embeddings", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败：%w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败：%w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
			} `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, fmt.Errorf("API 错误（状态码 %d）：响应解码失败", resp.StatusCode)
		}
		return nil, fmt.Errorf("API 错误（状态码 %d）：%s - %s", resp.StatusCode, errResp.Error.Type, errResp.Error.Message)
	}

	var embResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, fmt.Errorf("响应解码失败：%w", err)
	}

	// 按索引排序以确保正确顺序
	embeddings := make([][]float32, len(embResp.Data))
	for _, item := range embResp.Data {
		if item.Index >= 0 && item.Index < len(embeddings) {
			embeddings[item.Index] = item.Embedding
		}
	}

	return embeddings, nil
}

// GenerateEmbedding 生成单个嵌入向量，带重试逻辑
func (c *Client) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := c.GenerateEmbeddings(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("未返回嵌入向量")
	}
	return embeddings[0], nil
}

// GenerateWithRetry 使用指数退避重试生成嵌入向量
func (c *Client) GenerateWithRetry(ctx context.Context, texts []string, maxRetries int) ([][]float32, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		embeddings, err := c.GenerateEmbeddings(ctx, texts)
		if err == nil {
			return embeddings, nil
		}

		lastErr = err

		// 最后一次尝试后不等待
		if attempt < maxRetries {
			waitTime := time.Duration(1<<uint(attempt)) * time.Second // 1s, 2s, 4s, ...
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(waitTime):
				// 继续下一次重试
			}
		}
	}

	return nil, fmt.Errorf("所有重试均失败：%w", lastErr)
}

// GetModel 返回当前使用的模型名称
func (c *Client) GetModel() string {
	return c.model
}

// GetDimension 返回当前模型的嵌入维度
func (c *Client) GetDimension() int {
	// text-embedding-3-small 支持可配置的维度
	// 默认为 1536
	switch c.model {
	case "text-embedding-3-small":
		return 1536
	case "text-embedding-3-large":
		return 3072
	case "text-embedding-ada-002":
		return 1536
	default:
		return 1536
	}
}
