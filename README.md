# myRAG - Multi-user RAG System

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://go.dev/)

A high-performance, self-hosted RAG (Retrieval-Augmented Generation) system with knowledge graph support.

## Features

- 📚 **Knowledge Bases** - Create isolated knowledge bases per user/tenant
- 📄 **Multi-format Support** - PDF, DOCX, XLSX, Markdown, CSV, TXT
- 💬 **AI Chat** - Chat with your documents, get answers with citations
- 🔍 **Hybrid Search** - Vector + keyword search with optional GraphRAG
- 👥 **Multi-tenant** - Row-level security for data isolation
- 🚀 **High Performance** - Go backend with async document processing
- 🐳 **One-click Deploy** - Docker Compose setup

## Quick Start

```bash
# Clone the repository
git clone https://github.com/njujiangxiang/myRAG.git
cd myRAG

# Start all services
docker-compose up -d

# Access the web UI
open http://localhost:3000
```

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   React     │────▶│   Go API    │────▶│  PostgreSQL │
│   Frontend  │     │   Backend   │     │  (Metadata) │
└─────────────┘     └──────┬──────┘     └─────────────┘
                           │
                    ┌──────▼──────┐
                    │   NATS JS   │
                    │  (Queue)    │
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │   Workers   │
                    └──────┬──────┘
                           │
         ┌─────────────────┼─────────────────┐
         ▼                 ▼                 ▼
┌─────────────┐    ┌─────────────┐   ┌─────────────┐
│   Qdrant    │    │    MinIO    │   │  LLM APIs   │
│  (Vectors)  │    │   (Files)   │   │ (OpenAI)    │
└─────────────┘    └─────────────┘   └─────────────┘
```

## Tech Stack

- **Backend:** Go 1.21+ with Gin framework
- **Frontend:** React + TypeScript + shadcn/ui + Tailwind CSS
- **Database:** PostgreSQL 16
- **Vector DB:** Qdrant 1.7+
- **Message Queue:** NATS JetStream
- **File Storage:** MinIO (S3-compatible)
- **LLM:** OpenAI/Anthropic/Local (ollama)

## Configuration

Copy `.env.example` to `.env` and configure:

```env
# Database
DATABASE_URL=postgres://user:pass@localhost:5432/ragdb

# Qdrant
QDRANT_URL=http://localhost:6333

# NATS
NATS_URL=nats://localhost:4222

# MinIO
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin

# LLM
OPENAI_API_KEY=sk-...
```

## Development

```bash
# Run backend
go run cmd/server/main.go

# Run frontend (in web/ directory)
npm run dev

# Run tests
go test ./...
```

## Roadmap

- [x] v1.0: Core RAG functionality (vector search + chat)
- [ ] v1.1: GraphRAG integration (knowledge graph + entity extraction)
- [x] v1.2: Advanced search (hybrid search + BM25 + reranking)
- [ ] v1.3: Frontend UI (multi-tenant dashboard)
- [ ] v1.4: SSE streaming for chat responses
- [ ] v2.0: Advanced permissions and team collaboration

## License

MIT
