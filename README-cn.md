# myRAG - 多用户知识库问答系统

> 高性能、可自部署的 RAG（检索增强生成）系统，支持知识图谱和混合检索。

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://go.dev/)

---

## 🌟 项目简介

myRAG 是一个企业级的知识库问答系统，基于 Go + React 构建。它能够将您的文档（PDF、Word、Excel 等）转换为可对话的知识库，通过 AI 大模型提供准确的答案和引用。

**核心场景**：
- 📚 企业知识库管理
- 💬 智能文档问答
- 🔍 混合检索（向量 + 关键词 + 知识图谱）
- 👥 多租户数据隔离

---

## ✨ 主要特性

| 特性 | 说明 |
|------|------|
| 📚 **知识库管理** | 支持按用户/租户创建独立知识库，数据完全隔离 |
| 📄 **多格式支持** | PDF、DOCX、XLSX、Markdown、CSV、TXT 等主流格式 |
| 💬 **AI 智能对话** | 与文档对话，获取带引用的精准答案 |
| 🔍 **混合检索** | 向量检索 + 关键词检索 + GraphRAG 知识图谱 |
| 👥 **多租户架构** | 行级安全隔离，支持多用户/多团队 |
| 🚀 **高性能** | Go 后端 + 异步文档处理 + 消息队列 |
| 🐳 **一键部署** | Docker Compose 完整编排 |
| 📊 **重排序优化** | BGE Rerank 提升检索相关性 |

---

## 🚀 快速开始

### 方式一：Docker 部署（推荐）

```bash
# 克隆仓库
git clone https://github.com/njujiangxiang/myRAG.git
cd myRAG

# 启动所有服务
docker-compose up -d

# 访问 Web 界面
open http://localhost:3000
```

### 方式二：源码运行

```bash
# 运行后端
go run cmd/server/main.go

# 运行前端（在 web/ 目录下）
npm run dev

# 运行测试
go test ./...
```

---

## 🏗️ 系统架构

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   React     │────▶│   Go API    │────▶│  PostgreSQL │
│   前端界面   │     │   后端服务   │     │  元数据库    │
└─────────────┘     └──────┬──────┘     └─────────────┘
                           │
                    ┌──────▼──────┐
                    │   NATS JS   │
                    │  消息队列    │
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │   Workers   │
                    │  异步处理    │
                    └──────┬──────┘
                           │
         ┌─────────────────┼─────────────────┐
         ▼                 ▼                 ▼
┌─────────────┐    ┌─────────────┐   ┌─────────────┐
│   Qdrant    │    │    MinIO    │   │  LLM APIs   │
│  向量数据库  │    │  文件存储    │   │ 大模型接口   │
└─────────────┘    └─────────────┘   └─────────────┘
```

---

## 🛠️ 技术栈

| 层级 | 技术 | 说明 |
|------|------|------|
| **后端** | Go 1.21+ / Gin | 高性能 API 服务 |
| **前端** | React / TypeScript / shadcn-ui / Tailwind CSS | 现代化 Web 界面 |
| **数据库** | PostgreSQL 16 | 元数据存储 |
| **向量库** | Qdrant 1.7+ | 向量相似度检索 |
| **消息队列** | NATS JetStream | 异步任务处理 |
| **文件存储** | MinIO | S3 兼容对象存储 |
| **大模型** | OpenAI / Anthropic / Ollama | 支持多种 LLM |
| **嵌入模型** | BGE / text-embedding | 中文优化嵌入 |
| **重排序** | BGE Rerank | 提升检索精度 |

---

## 📋 核心功能

### 1. 文档处理
- ✅ 多格式解析（PDF/Word/Excel/Markdown 等）
- ✅ 智能文本分块（按段落/标题/固定长度）
- ✅ 异步处理队列（不阻塞主流程）
- ✅ 处理进度实时追踪

### 2. 检索增强
- ✅ 向量检索（语义相似度）
- ✅ BM25 关键词检索
- ✅ 混合检索（加权融合）
- ✅ BGE Rerank 重排序
- ✅ GraphRAG 知识图谱检索（开发中）

### 3. 对话系统
- ✅ 多轮对话上下文
- ✅ 答案引用溯源
- ✅ SSE 流式响应
- ✅ 对话历史管理

### 4. 多租户
- ✅ 用户注册/登录（JWT 认证）
- ✅ 知识库隔离
- ✅ 行级安全策略
- ✅ 团队协作支持（规划中）

---

## ⚙️ 配置说明

复制 `.env.example` 到 `.env` 并配置：

```env
# 数据库配置
DATABASE_URL=postgres://user:pass@localhost:5432/ragdb

# Qdrant 向量库
QDRANT_URL=http://localhost:6333

# NATS 消息队列
NATS_URL=nats://localhost:4222

# MinIO 文件存储
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin

# 大模型配置
OPENAI_API_KEY=sk-...
# 或使用本地模型
OLLAMA_BASE_URL=http://localhost:11434

# JWT 密钥
JWT_SECRET=your-secret-key
```

---

## 📂 项目结构

```
myrag/
├── cmd/                    # 应用程序入口
│   └── server/            # 主服务
├── internal/              # 内部包
│   ├── config/           # 配置管理
│   ├── database/         # 数据库连接
│   ├── embedding/        # 嵌入模型客户端
│   ├── handler/          # HTTP 处理器
│   ├── jwt/              # JWT 认证
│   ├── minio/            # MinIO 客户端
│   ├── models/           # 数据模型
│   ├── nats/             # NATS 客户端
│   ├── parser/           # 文档解析器
│   ├── qdrant/           # Qdrant 客户端
│   ├── rag/              # RAG 核心逻辑
│   ├── rerank/           # 重排序服务
│   └── worker/           # 后台任务处理
├── migrations/            # 数据库迁移
├── services/              # 外部服务配置
├── web/                   # 前端代码
├── docker-compose.yml     # Docker 编排
└── start.sh              # 启动脚本
```

---

## 📅 开发路线图

| 版本 | 状态 | 功能 |
|------|------|------|
| v1.0 | ✅ 已完成 | 核心 RAG 功能（向量检索 + 对话） |
| v1.1 | 🔄 开发中 | GraphRAG 集成（知识图谱 + 实体抽取） |
| v1.2 | ✅ 已完成 | 高级检索（混合检索 + BM25 + 重排序） |
| v1.3 | 🔄 开发中 | 前端界面（多租户仪表板） |
| v1.4 | 📋 计划中 | SSE 流式响应 |
| v2.0 | 📋 计划中 | 高级权限和团队协作 |

---

## 🔧 常用命令

```bash
# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f server

# 重启服务
docker-compose restart

# 清理并重新部署
docker-compose down -v && docker-compose up -d

# 运行测试
go test ./...

# 构建二进制
go build -o server cmd/server/main.go
```

---

## 📄 相关文档

| 文档 | 说明 |
|------|------|
| [DESIGN.md](DESIGN.md) | 系统设计文档 |
| [STARTUP.md](STARTUP.md) | 启动和部署指南 |
| [MULTI_RAG.md](MULTI_RAG.md) | 多路检索实现说明 |
| [BGE_RERANK_SUMMARY.md](BGE_RERANK_SUMMARY.md) | BGE 重排序模型说明 |
| [GRAPH_RAG_CHINESE_COMMENTS.md](GRAPH_RAG_CHINESE_COMMENTS.md) | GraphRAG 中文注释 |
| [CODE_REVIEW.md](CODE_REVIEW.md) | 代码审查报告 |
| [TEST_REPORT.md](TEST_REPORT.md) | 测试报告 |

---

## 👨‍💻 作者

**江翔** (njujiangxiang)

- GitHub: [@njujiangxiang](https://github.com/njujiangxiang)
- 邮箱：nju.jiangxiang@gmail.com

---

<div align="center">

Made with ❤️ by 江翔

</div>
