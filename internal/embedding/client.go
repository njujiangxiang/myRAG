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

// Client handles embedding generation
type Client struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// Config holds embedding client configuration
type Config struct {
	APIKey  string
	BaseURL string
	Model   string
}

// DefaultConfig returns default configuration
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

// New creates a new embedding client
func New(config Config) *Client {
	return &Client{
		apiKey:  config.APIKey,
		baseURL: config.BaseURL,
		model:   config.Model,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// EmbeddingRequest represents an OpenAI embedding request
type EmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// EmbeddingResponse represents an OpenAI embedding response
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

// GenerateEmbeddings generates embeddings for a batch of texts
func (c *Client) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	// OpenAI API supports up to 2048 batch size
	// But we'll use a smaller batch size for reliability
	batchSize := 100
	var allEmbeddings [][]float32

	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}

		batch := texts[i:end]
		embeddings, err := c.generateBatch(ctx, batch)
		if err != nil {
			return nil, fmt.Errorf("failed to generate batch starting at index %d: %w", i, err)
		}

		allEmbeddings = append(allEmbeddings, embeddings...)
	}

	return allEmbeddings, nil
}

// generateBatch generates embeddings for a single batch
func (c *Client) generateBatch(ctx context.Context, texts []string) ([][]float32, error) {
	reqBody := EmbeddingRequest{
		Model: c.model,
		Input: texts,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/embeddings", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
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
			return nil, fmt.Errorf("API error (status %d): failed to decode response", resp.StatusCode)
		}
		return nil, fmt.Errorf("API error (status %d): %s - %s", resp.StatusCode, errResp.Error.Type, errResp.Error.Message)
	}

	var embResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Sort by index to ensure correct order
	embeddings := make([][]float32, len(embResp.Data))
	for _, item := range embResp.Data {
		if item.Index >= 0 && item.Index < len(embeddings) {
			embeddings[item.Index] = item.Embedding
		}
	}

	return embeddings, nil
}

// GenerateEmbedding generates a single embedding with retry logic
func (c *Client) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := c.GenerateEmbeddings(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	return embeddings[0], nil
}

// GenerateWithRetry generates embeddings with exponential backoff retry
func (c *Client) GenerateWithRetry(ctx context.Context, texts []string, maxRetries int) ([][]float32, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		embeddings, err := c.GenerateEmbeddings(ctx, texts)
		if err == nil {
			return embeddings, nil
		}

		lastErr = err

		// Don't wait after the last attempt
		if attempt < maxRetries {
			waitTime := time.Duration(1<<uint(attempt)) * time.Second // 1s, 2s, 4s, ...
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(waitTime):
				// Continue to next retry
			}
		}
	}

	return nil, fmt.Errorf("all retries failed: %w", lastErr)
}

// GetModel returns the current model name
func (c *Client) GetModel() string {
	return c.model
}

// GetDimension returns the embedding dimension for the current model
func (c *Client) GetDimension() int {
	// text-embedding-3-small supports configurable dimensions
	// Default is 1536
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
