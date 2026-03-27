# BGE Rerank 配置指南

## 概述

Rerank（重排序）功能使用 **BGE Cross-Encoder 模型** 对初步召回的搜索结果进行语义相关性重排序，显著提升搜索质量。

本项目使用 **自托管 BGE 服务**（基于 FlagEmbedding），无需依赖外部 API，数据完全内网可控。

## 架构

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   myRAG Server  │────▶│ RerankClient    │────▶│ BGE FastAPI     │
│   (Go)          │     │ (internal/rerank)│    │ Service (Python)│
└─────────────────┘     └─────────────────┘     └─────────────────┘
                                                       │
                                                       ▼
                                              ┌─────────────────┐
                                              │ BGE-Reranker    │
                                              │ (FlagEmbedding) │
                                              └─────────────────┘
```

## 快速开始

### 1. 启动 BGE Rerank 服务

**使用 Docker（推荐）**：

```bash
cd services/bge-rerank

# GPU 模式（需要 NVIDIA GPU）
docker-compose up -d

# CPU 模式（编辑 docker-compose.yml，修改 BGE_DEVICE=cpu）
docker-compose up -d
```

**手动启动**：

```bash
cd services/bge-rerank

# 安装依赖
pip install -r requirements.txt

# 启动服务
python main.py
```

服务默认运行在 `http://localhost:8800`

### 2. 配置 myRAG

在 `.env` 文件中：

```bash
# 启用 Rerank
BGE_RERANK_ENABLED=true

# BGE 服务地址
BGE_RERANK_BASE_URL=http://localhost:8800

# 模型配置
BGE_RERANK_MODEL=BAAI/bge-reranker-v2-m3

# 结果数量
BGE_RERANK_TOP_K=10
BGE_RERANK_CANDIDATES=50
```

### 3. 创建知识库

创建知识库时设置 `rag_type="rerank"`：

```bash
curl -X POST http://localhost:8080/api/v1/kbs \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "name": "My Rerank KB",
    "description": "Knowledge base with BGE reranking",
    "rag_type": "rerank"
  }'
```

## BGE 模型选择

| 模型 | 语言 | 速度 | 质量 | 推荐场景 |
|------|------|------|------|----------|
| `BAAI/bge-reranker-v2-m3` | 多语言 | 中 | 最高 | 生产环境，多语言支持 |
| `BAAI/bge-reranker-v2-base` | 多语言 | 中 | 高 | 平衡速度与质量 |
| `BAAI/bge-reranker-v2-minico` | 多语言 | 快 | 中 | 低延迟场景 |

### 修改模型

在 `docker-compose.yml` 中修改：

```yaml
environment:
  - BGE_MODEL=BAAI/bge-reranker-v2-minico
```

## 性能参考

| 模式 | 延迟/查询 | GPU 内存 | 推荐 |
|------|-----------|----------|------|
| v2-m3 (GPU) | ~50ms | 6GB | 生产环境 |
| v2-base (GPU) | ~30ms | 4GB | 平衡场景 |
| v2-minico (GPU) | ~15ms | 2GB | 低延迟 |
| v2-m3 (CPU) | ~500ms | N/A | 开发测试 |

## API 参考

### 健康检查

```bash
curl http://localhost:8800/health
```

响应：
```json
{"status": "healthy", "model": "BAAI/bge-reranker-v2-m3"}
```

### Rerank 请求

```bash
curl -X POST http://localhost:8800/rerank \
  -H "Content-Type: application/json" \
  -d '{
    "query": "人工智能发展",
    "documents": ["文档 1 内容", "文档 2 内容"],
    "top_n": 10
  }'
```

响应：
```json
{
  "results": [
    {"index": 1, "score": 0.95, "text": "文档 2 内容"},
    {"index": 0, "score": 0.72, "text": "文档 1 内容"}
  ]
}
```

## 故障排除

### 服务启动失败

**问题**: `FlagEmbedding not installed`

**解决**:
```bash
pip install -r requirements.txt
# 或
pip install FlagEmbedding
```

### 模型下载慢

BGE 模型从 HuggingFace 下载，国内可能较慢。

**解决** - 使用镜像：
```bash
export HF_ENDPOINT=https://hf-mirror.com
docker-compose up -d
```

### GPU 不可用

**问题**: `CUDA out of memory` 或 `no CUDA device`

**解决** - 切换到 CPU 模式：
```yaml
# docker-compose.yml
environment:
  - BGE_DEVICE=cpu
```

或减少 batch size。

### myRAG 连接失败

**检查** BGE 服务是否运行：
```bash
curl http://localhost:8800/health
```

确保 `BGE_RERANK_BASE_URL` 配置正确。

## 部署建议

### 生产环境

1. **GPU 部署**: 至少 1 张 T4 或同等 GPU
2. **多实例**: 使用负载均衡分发请求
3. **缓存**: 对高频查询结果缓存
4. **监控**: 添加 Prometheus 监控指标

### 开发环境

1. 使用 CPU 模式即可
2. 选择 `v2-minico` 模型加快速度
3. 减少 `CANDIDATES` 数量降低负载

## 成本对比

| 方案 | 成本 | 延迟 | 数据隐私 |
|------|------|------|----------|
| **BGE 自托管** | 免费（仅硬件） | ~50ms | 内网 |
| Jina AI API | $0.02/1k 文档 | ~200ms | 外网 |
| Cohere API | $0.09/1k 文档 | ~150ms | 外网 |

**回本周期**: 约 50 万次请求后，自托管成本低于 API 调用。
