# BGE Rerank Service

自托管的 BGE Cross-Encoder 重排序服务，用于提升 RAG 搜索相关性。

## 快速开始

### Docker 部署（推荐）

```bash
# GPU 模式
docker-compose up -d

# CPU 模式（编辑 docker-compose.yml 修改 BGE_DEVICE=cpu）
docker-compose up -d
```

### 手动部署

```bash
# 安装依赖
pip install -r requirements.txt

# 启动服务
python main.py
```

服务运行在 `http://localhost:8800`

## API 端点

### 健康检查
```bash
curl http://localhost:8800/health
```

### Rerank
```bash
curl -X POST http://localhost:8800/rerank \
  -H "Content-Type: application/json" \
  -d '{"query": "查询", "documents": ["文档 1", "文档 2"], "top_n": 10}'
```

## 配置

| 环境变量 | 默认值 | 说明 |
|----------|--------|------|
| `BGE_MODEL` | `BAAI/bge-reranker-v2-m3` | 模型名称 |
| `BGE_DEVICE` | `cuda` | `cuda` 或 `cpu` |
| `BGE_HOST` | `0.0.0.0` | 监听地址 |
| `BGE_PORT` | `8800` | 监听端口 |

## 模型选择

- **v2-m3**: 最佳质量，多语言支持（推荐）
- **v2-base**: 平衡速度与质量
- **v2-minico**: 最快，适合低延迟场景

## 资源需求

| 模型 | GPU 内存 | CPU 内存 |
|------|----------|----------|
| v2-m3 | 6GB | 8GB |
| v2-base | 4GB | 4GB |
| v2-minico | 2GB | 2GB |
