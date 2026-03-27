package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
)

// Config holds all configuration
type Config struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	Qdrant   QdrantConfig   `json:"qdrant"`
	NATS     NATSConfig     `json:"nats"`
	MinIO    MinIOConfig    `json:"minio"`
	LLM      LLMConfig      `json:"llm"`
	JWT      JWTConfig      `json:"jwt"`
	Rerank   RerankConfig   `json:"rerank"`
}

// RerankConfig holds rerank configuration for BGE self-hosted service
type RerankConfig struct {
	Enabled    bool   `json:"enabled"`
	BaseURL    string `json:"base_url"` // BGE service URL
	Model      string `json:"model"`    // Model name (used by BGE service)
	TopK       int    `json:"top_k"`
	Candidates int    `json:"candidates"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port string `json:"port"`
	Env  string `json:"env"` // development, staging, production
}

// DatabaseConfig holds PostgreSQL configuration
type DatabaseConfig struct {
	URL string `json:"url"`
}

// QdrantConfig holds Qdrant vector DB configuration
type QdrantConfig struct {
	URL    string `json:"url"`
	APIKey string `json:"api_key"`
}

// NATSConfig holds NATS JetStream configuration
type NATSConfig struct {
	URL string `json:"url"`
}

// MinIOConfig holds MinIO S3 configuration
type MinIOConfig struct {
	Endpoint  string `json:"endpoint"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Bucket    string `json:"bucket"`
	UseSSL    bool   `json:"use_ssl"`
}

// LLMConfig holds LLM provider configuration
type LLMConfig struct {
	Provider string `json:"provider"` // openai, anthropic, local
	APIKey   string `json:"api_key"`
	Model    string `json:"model"` // e.g., text-embedding-3-small
}

// JWTConfig holds JWT authentication configuration
type JWTConfig struct {
	Secret string        `json:"secret"`
	Expiry time.Duration `json:"expiry"`
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	k := koanf.New(".")

	// Load from environment variables with MYRAG_ prefix
	// e.g., MYRAG_SERVER_PORT=8080
	if err := k.Load(env.Provider("MYRAG_", ".", func(s string) string {
		// MYRAG_SERVER_PORT -> server.port
		return s[5:]
	}), nil); err != nil {
		return nil, fmt.Errorf("failed to load env config: %w", err)
	}

	cfg := &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
			Env:  getEnv("SERVER_ENV", "development"),
		},
		Database: DatabaseConfig{
			URL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/ragdb?sslmode=disable"),
		},
		Qdrant: QdrantConfig{
			URL:    getEnv("QDRANT_URL", "http://localhost:6333"),
			APIKey: getEnv("QDRANT_API_KEY", ""),
		},
		NATS: NATSConfig{
			URL: getEnv("NATS_URL", "nats://localhost:4222"),
		},
		MinIO: MinIOConfig{
			Endpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKey: getEnv("MINIO_ACCESS_KEY", "minioadmin"),
			SecretKey: getEnv("MINIO_SECRET_KEY", "minioadmin"),
			Bucket:    getEnv("MINIO_BUCKET", "documents"),
			UseSSL:    getEnv("MINIO_USE_SSL", "false") == "true",
		},
		LLM: LLMConfig{
			Provider: getEnv("LLM_PROVIDER", "openai"),
			APIKey:   getEnv("OPENAI_API_KEY", ""),
			Model:    getEnv("OPENAI_MODEL", "text-embedding-3-small"),
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", ""),
			Expiry: 24 * time.Hour,
		},
		Rerank: RerankConfig{
			Enabled:    getEnv("BGE_RERANK_ENABLED", "false") == "true",
			BaseURL:    getEnv("BGE_RERANK_BASE_URL", "http://localhost:8800"),
			Model:      getEnv("BGE_RERANK_MODEL", "BAAI/bge-reranker-v2-m3"),
			TopK:       getEnvInt("BGE_RERANK_TOP_K", 10),
			Candidates: getEnvInt("BGE_RERANK_CANDIDATES", 50),
		},
	}

	// Validate required configuration
	if cfg.LLM.APIKey == "" && cfg.LLM.Provider != "local" {
		return nil, fmt.Errorf("LLM API key is required for %s provider", cfg.LLM.Provider)
	}

	// JWT secret is required in production
	if cfg.JWT.Secret == "" && cfg.Server.Env == "production" {
		return nil, fmt.Errorf("JWT_SECRET environment variable is required in production")
	}

	// Generate JWT secret for development if not set
	if cfg.JWT.Secret == "" {
		cfg.JWT.Secret = generateDevSecret()
	}

	return cfg, nil
}

// getEnv gets environment variable with fallback
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// getEnvInt gets integer environment variable with fallback
func getEnvInt(key string, fallback int) int {
	if value, exists := os.LookupEnv(key); exists {
		if result, err := strconv.Atoi(value); err == nil {
			return result
		}
	}
	return fallback
}

// generateDevSecret generates a random secret for development
func generateDevSecret() string {
	// Simple dev secret - not for production use
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("dev-secret-%d", time.Now().UnixNano())))
}
