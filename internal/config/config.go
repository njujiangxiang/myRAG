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

// Config 持有所有配置
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

// RerankConfig 持有 BGE 自托管重排序服务的配置
type RerankConfig struct {
	Enabled    bool   `json:"enabled"`
	BaseURL    string `json:"base_url"` // BGE 服务地址
	Model      string `json:"model"`    // 模型名称（由 BGE 服务使用）
	TopK       int    `json:"top_k"`
	Candidates int    `json:"candidates"`
}

// ServerConfig 持有 HTTP 服务器配置
type ServerConfig struct {
	Port string `json:"port"`
	Env  string `json:"env"` // development, staging, production
}

// DatabaseConfig 持有 PostgreSQL 配置
type DatabaseConfig struct {
	URL string `json:"url"`
}

// QdrantConfig 持有 Qdrant 向量数据库配置
type QdrantConfig struct {
	URL    string `json:"url"`
	APIKey string `json:"api_key"`
}

// NATSConfig 持有 NATS JetStream 配置
type NATSConfig struct {
	URL string `json:"url"`
}

// MinIOConfig 持有 MinIO S3 配置
type MinIOConfig struct {
	Endpoint  string `json:"endpoint"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	Bucket    string `json:"bucket"`
	UseSSL    bool   `json:"use_ssl"`
}

// LLMConfig 持有 LLM 提供商配置
type LLMConfig struct {
	Provider string `json:"provider"` // openai, anthropic, local
	APIKey   string `json:"api_key"`
	Model    string `json:"model"` // 例如：text-embedding-3-small
}

// JWTConfig 持有 JWT 认证配置
type JWTConfig struct {
	Secret string        `json:"secret"`
	Expiry time.Duration `json:"expiry"`
}

// Load 从环境变量读取配置
func Load() (*Config, error) {
	k := koanf.New(".")

	// 从环境变量加载，使用 MYRAG_ 前缀
	// 例如：MYRAG_SERVER_PORT=8080
	if err := k.Load(env.Provider("MYRAG_", ".", func(s string) string {
		// MYRAG_SERVER_PORT -> server.port
		return s[5:]
	}), nil); err != nil {
		return nil, fmt.Errorf("加载环境变量配置失败：%w", err)
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

	// 验证必需的配置
	if cfg.LLM.APIKey == "" && cfg.LLM.Provider != "local" {
		return nil, fmt.Errorf("%s 提供商需要 LLM API 密钥", cfg.LLM.Provider)
	}

	// 生产环境需要 JWT 密钥
	if cfg.JWT.Secret == "" && cfg.Server.Env == "production" {
		return nil, fmt.Errorf("生产环境需要 JWT_SECRET 环境变量")
	}

	// 开发环境如果没有设置，生成随机密钥
	if cfg.JWT.Secret == "" {
		cfg.JWT.Secret = generateDevSecret()
	}

	return cfg, nil
}

// getEnv 获取环境变量，带默认值回退
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// getEnvInt 获取整数类型环境变量，带默认值回退
func getEnvInt(key string, fallback int) int {
	if value, exists := os.LookupEnv(key); exists {
		if result, err := strconv.Atoi(value); err == nil {
			return result
		}
	}
	return fallback
}

// generateDevSecret 生成开发环境随机密钥
func generateDevSecret() string {
	// 简单的开发密钥 - 不用于生产环境
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("dev-secret-%d", time.Now().UnixNano())))
}
