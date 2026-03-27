package rerank

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
)

// Client 通过 BGE Cross-Encoder 模型处理重排序
// 兼容 BGE FastAPI 服务 (https://github.com/FlagOpen/FlagEmbedding)
type Client struct {
	baseURL    string
	model      string
	topK       int
	httpClient *http.Client
}

// Config 持有重排序客户端配置
type Config struct {
	BaseURL string // BGE 服务地址，如 http://localhost:8800
	Model   string // 模型名称（由 BGE 服务使用）
	TopK    int    // 重排序后返回的结果数量
}

// DefaultConfig 返回 BGE 默认配置
func DefaultConfig() Config {
	baseURL := getEnv("BGE_RERANK_BASE_URL", "http://localhost:8800")
	model := getEnv("BGE_RERANK_MODEL", "BAAI/bge-reranker-v2-m3")
	topK := getEnvInt("BGE_RERANK_TOP_K", 10)

	return Config{
		BaseURL: baseURL,
		Model:   model,
		TopK:    topK,
	}
}

// New 创建新的重排序客户端
func New(config Config) *Client {
	return &Client{
		baseURL: config.BaseURL,
		model:   config.Model,
		topK:    config.TopK,
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // 自托管服务需要更长的超时时间
		},
	}
}

// RerankRequest 表示 BGE 重排序 API 请求
type RerankRequest struct {
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
	TopN      int      `json:"top_n,omitempty"`
}

// RerankResponse 表示 BGE 重排序 API 响应
type RerankResponse struct {
	Results []RerankResult `json:"results"`
}

// RerankResult 表示单个重排序结果
type RerankResult struct {
	Index int     `json:"index"`
	Score float32 `json:"score"`
	Text  string  `json:"text,omitempty"`
}

// Rerank 使用 BGE 服务对文档列表进行重排序
// 返回按相关性分数排序的结果
func (c *Client) Rerank(ctx context.Context, query string, documents []string, topN int) ([]RerankResult, error) {
	if len(documents) == 0 {
		return []RerankResult{}, nil
	}

	// 限制文档数量以防止请求过大
	const maxDocuments = 1000
	if len(documents) > maxDocuments {
		return nil, fmt.Errorf("文档数量过多：%d，最大允许 %d", len(documents), maxDocuments)
	}

	// 如果未指定 topN，使用配置的 TopK
	if topN <= 0 {
		topN = c.topK
	}

	reqBody := RerankRequest{
		Query:     query,
		Documents: documents,
		TopN:      topN,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败：%w", err)
	}

	// BGE 服务端点：POST /rerank
	url := c.baseURL + "/rerank"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败：%w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// 带重试逻辑的执行请求，提高网络弹性
	var resp *http.Response
	var retryErr error
	maxRetries := 3

	for attempt := 0; attempt < maxRetries; attempt++ {
		resp, retryErr = c.httpClient.Do(req)
		if retryErr == nil {
			break
		}

		// 最后一次尝试后不等待
		if attempt < maxRetries-1 {
			wait := time.Duration(1<<uint(attempt)) * time.Second // 1 秒，2 秒，4 秒
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(wait):
			}
		}
	}

	if retryErr != nil {
		return nil, fmt.Errorf("重试 %d 次后请求失败：%w", maxRetries, retryErr)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, fmt.Errorf("API 错误（状态码 %d）：响应解码失败", resp.StatusCode)
		}
		return nil, fmt.Errorf("API 错误（状态码 %d）：%s", resp.StatusCode, errResp.Error)
	}

	var rerankResp RerankResponse
	if err := json.NewDecoder(resp.Body).Decode(&rerankResp); err != nil {
		return nil, fmt.Errorf("响应解码失败：%w", err)
	}

	return rerankResp.Results, nil
}

// HealthCheck 检查 BGE 服务是否可用
func (c *Client) HealthCheck(ctx context.Context) error {
	url := c.baseURL + "/health"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("创建健康检查请求失败：%w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("健康检查失败：%w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("服务不健康：状态码 %d", resp.StatusCode)
	}

	return nil
}

// GetTopK 返回配置的 TopK 值
func (c *Client) GetTopK() int {
	return c.topK
}

// getEnv 获取环境变量，带默认值回退
func getEnv(key, fallback string) string {
	if value := getEnvValue(key); value != "" {
		return value
	}
	return fallback
}

// getEnvInt 获取整数类型环境变量，带默认值回退
func getEnvInt(key string, fallback int) int {
	if value := getEnvValue(key); value != "" {
		if result, err := strconv.Atoi(value); err == nil {
			return result
		}
	}
	return fallback
}

// getEnvValue 获取环境变量值
func getEnvValue(key string) string {
	return os.Getenv(key)
}
