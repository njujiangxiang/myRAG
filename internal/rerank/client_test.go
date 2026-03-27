package rerank

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Mock BGE server for testing
func newMockBGEServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"status": "healthy",
			})

		case "/rerank":
			var req struct {
				Query     string   `json:"query"`
				Documents []string `json:"documents"`
				TopN      int      `json:"top_n"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			// Mock rerank scores based on document length
			results := make([]map[string]interface{}, len(req.Documents))
			for i, doc := range req.Documents {
				// Simulate score based on query match (mock logic)
				score := 0.5 + float64(len(doc)%10)/20.0
				results[i] = map[string]interface{}{
					"index": i,
					"score": score,
					"text":  doc,
				}
			}

			// Sort by score descending (mock already sorted)
			for i := 0; i < len(results)-1; i++ {
				for j := i + 1; j < len(results); j++ {
					if results[i]["score"].(float64) < results[j]["score"].(float64) {
						results[i], results[j] = results[j], results[i]
					}
				}
			}

			// Apply top_n
			topN := req.TopN
			if topN <= 0 || topN > len(results) {
				topN = len(results)
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"results": results[:topN],
			})

		default:
			http.NotFound(w, r)
		}
	}))
}

func TestClient_Rerank(t *testing.T) {
	server := newMockBGEServer()
	defer server.Close()

	client := New(Config{
		BaseURL: server.URL,
		Model:   "test-model",
		TopK:    5,
	})

	ctx := context.Background()

	t.Run("successful rerank", func(t *testing.T) {
		docs := []string{"doc 1", "doc 2", "doc 3"}
		results, err := client.Rerank(ctx, "test query", docs, 2)
		if err != nil {
			t.Fatalf("Rerank failed: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("Expected 2 results, got %d", len(results))
		}

		// Check scores are in descending order
		for i := 1; i < len(results); i++ {
			if results[i].Score > results[i-1].Score {
				t.Errorf("Results not sorted: [%d].score > [%d].score", i, i-1)
			}
		}
	})

	t.Run("empty documents", func(t *testing.T) {
		results, err := client.Rerank(ctx, "test query", []string{}, 5)
		if err != nil {
			t.Fatalf("Rerank failed: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("Expected 0 results for empty input, got %d", len(results))
		}
	})

	t.Run("uses topK when topN is 0", func(t *testing.T) {
		docs := make([]string, 10)
		for i := range docs {
			docs[i] = "doc"
		}
		results, err := client.Rerank(ctx, "test", docs, 0)
		if err != nil {
			t.Fatalf("Rerank failed: %v", err)
		}
		if len(results) != client.topK {
			t.Errorf("Expected %d results (topK), got %d", client.topK, len(results))
		}
	})
}

func TestClient_HealthCheck(t *testing.T) {
	server := newMockBGEServer()
	defer server.Close()

	client := New(Config{
		BaseURL: server.URL,
		TopK:    5,
	})

	ctx := context.Background()

	t.Run("healthy service", func(t *testing.T) {
		err := client.HealthCheck(ctx)
		if err != nil {
			t.Fatalf("Health check failed: %v", err)
		}
	})

	t.Run("unhealthy service", func(t *testing.T) {
		client := New(Config{
			BaseURL: "http://invalid-host",
			TopK:    5,
		})
		err := client.HealthCheck(ctx)
		if err == nil {
			t.Fatal("Expected health check to fail")
		}
	})
}

func TestClient_Timeout(t *testing.T) {
	// Create slow server
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer slowServer.Close()

	client := New(Config{
		BaseURL: slowServer.URL,
		TopK:    5,
	})

	// Set short timeout
	client.httpClient.Timeout = 100 * time.Millisecond

	ctx := context.Background()
	_, err := client.Rerank(ctx, "test", []string{"doc"}, 5)

	if err == nil {
		t.Fatal("Expected timeout error")
	}
}

func TestGetTopK(t *testing.T) {
	client := New(Config{
		BaseURL: "http://localhost",
		TopK:    15,
	})

	if client.GetTopK() != 15 {
		t.Errorf("Expected TopK=15, got %d", client.GetTopK())
	}
}

func TestDefaultConfig(t *testing.T) {
	t.Setenv("BGE_RERANK_BASE_URL", "http://test:8800")
	t.Setenv("BGE_RERANK_MODEL", "test-model")
	t.Setenv("BGE_RERANK_TOP_K", "20")

	cfg := DefaultConfig()

	if cfg.BaseURL != "http://test:8800" {
		t.Errorf("BaseURL = %s, want http://test:8800", cfg.BaseURL)
	}
	if cfg.Model != "test-model" {
		t.Errorf("Model = %s, want test-model", cfg.Model)
	}
	if cfg.TopK != 20 {
		t.Errorf("TopK = %d, want 20", cfg.TopK)
	}
}
