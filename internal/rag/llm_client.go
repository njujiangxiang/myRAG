package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LLMClient 封装 LLM API 调用，用于实体提取
// 支持 OpenAI 和 Anthropic 两种 LLM 提供商
type LLMClient struct {
	httpClient *http.Client // HTTP 客户端
	apiKey     string       // API 密钥
	baseURL    string       // API 基础 URL
	model      string       // 模型名称
	provider   string       // 提供商名称（openai/anthropic）
}

// NewLLMClient 创建一个新的 LLM 客户端
// 参数:
//   - apiKey: LLM API 密钥
//   - model: 使用的模型名称
//   - provider: 提供商名称（"openai" 或 "anthropic"）
//   - baseURL: API 基础 URL（可选，为空时使用默认值）
func NewLLMClient(apiKey, model, provider, baseURL string) *LLMClient {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
		if provider == "anthropic" {
			baseURL = "https://api.anthropic.com/v1"
		}
	}

	return &LLMClient{
		httpClient: &http.Client{Timeout: 120 * time.Second}, // 120 秒超时
		apiKey:     apiKey,
		baseURL:    baseURL,
		model:      model,
		provider:   provider,
	}
}

// Generate 使用 LLM API 生成文本
// 根据提供商自动选择合适的 API 端点和格式
// 参数:
//   - ctx: 上下文
//   - prompt: 输入提示词
// 返回:
//   - string: LLM 生成的文本
//   - error: 错误信息
func (c *LLMClient) Generate(ctx context.Context, prompt string) (string, error) {
	if c.provider == "anthropic" {
		return c.generateAnthropic(ctx, prompt)
	}
	return c.generateOpenAI(ctx, prompt)
}

// generateOpenAI 使用 OpenAI 兼容 API 生成文本
// 适用于 OpenAI 及其他兼容 OpenAI 格式的 LLM 服务
func (c *LLMClient) generateOpenAI(ctx context.Context, prompt string) (string, error) {
	// 构建消息列表
	messages := []map[string]string{
		{"role": "system", "content": "You are a helpful assistant that extracts structured information from text. Only return JSON, no other text."},
		{"role": "user", "content": prompt},
	}

	// 构建请求体
	reqBody := map[string]any{
		"model":       c.model,
		"messages":    messages,
		"max_tokens":  2000,
		"temperature": 0.1, // 低温度值确保输出稳定
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	// 发送请求（带重试）
	resp, err := c.doRequestWithRetry(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// 解析响应
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

// generateAnthropic 使用 Anthropic API 生成文本
// Anthropic API 格式与 OpenAI 不同，需要特殊处理
func (c *LLMClient) generateAnthropic(ctx context.Context, prompt string) (string, error) {
	// 构建 Anthropic 请求体
	reqBody := map[string]any{
		"model":       c.model,
		"max_tokens":  2000,
		"system":      "You are a helpful assistant that extracts structured information from text. Only return JSON, no other text.",
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/messages", bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01") // Anthropic API 版本

	// 发送请求（带重试）
	resp, err := c.doRequestWithRetry(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Anthropic API error (status %d): %s", resp.StatusCode, string(body))
	}

	// 解析 Anthropic 响应格式
	var result struct {
		Content []struct {
			Text string `json:"text"`
			Type string `json:"type"`
		} `json:"content"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Content) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	return result.Content[0].Text, nil
}

// doRequestWithRetry 执行 HTTP 请求，带指数退避重试机制
// 用于处理网络错误和速率限制
// 参数:
//   - req: HTTP 请求
// 返回:
//   - *http.Response: HTTP 响应
//   - error: 错误信息
func (c *LLMClient) doRequestWithRetry(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	// 最多重试 3 次
	for attempt := 0; attempt < 3; attempt++ {
		resp, err = c.httpClient.Do(req)
		if err != nil {
			// 网络错误，等待后重试
			if attempt < 2 {
				time.Sleep(time.Duration(attempt+1) * time.Second)
				continue
			}
			return nil, fmt.Errorf("request failed after %d attempts: %w", attempt+1, err)
		}

		// 遇到速率限制或服务不可用，等待后重试
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
			resp.Body.Close()
			// 指数退避：2 秒、4 秒、6 秒
			backoff := time.Duration(attempt+1) * 2 * time.Second
			time.Sleep(backoff)
			continue
		}

		return resp, nil
	}

	return resp, err
}
