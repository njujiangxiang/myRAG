# BGE Rerank 实现总结

## 完成的工作

### 1. Go 客户端 (`internal/rerank/client.go`)
- 调用自托管 BGE 服务的 HTTP 客户端
- 支持健康检查 endpoint
- 兼容 BGE FastAPI 服务响应格式

### 2. Python BGE 服务 (`services/bge-rerank/`)
```
services/bge-rerank/
├── main.py              # FastAPI 服务主文件
├── requirements.txt     # Python 依赖
├── Dockerfile           # Docker 镜像配置
├── docker-compose.yml   # Docker Compose 部署
├── start.sh             # 本地启动脚本
├── test.sh              # 服务测试脚本
├── README.md            # 服务文档
└── DEPLOY.md            # 部署指南
```

### 3. 配置更新
- `internal/config/config.go` - BGE 配置结构
- `internal/rag/factory.go` - RerankConfig 更新
- `cmd/server/main.go` - 传递 BGE 配置
- `.env.example` - BGE 环境变量示例

### 4. 文档
- `docs/RERANK_CONFIG.md` - 完整配置和使用指南

## 启动服务

```bash
# 方式 1: 本地启动（需要 Python）
cd services/bge-rerank
./start.sh

# 方式 2: Docker 部署（需要 Docker）
cd services/bge-rerank
docker compose up -d
```

## 测试服务

```bash
cd services/bge-rerank
./test.sh
```

## 配置 myRAG

在 `.env` 文件中：
```bash
BGE_RERANK_ENABLED=true
BGE_RERANK_BASE_URL=http://localhost:8800
BGE_RERANK_TOP_K=10
BGE_RERANK_CANDIDATES=50
```

## 创建 Rerank 知识库

```bash
curl -X POST http://localhost:8080/api/v1/kbs \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer TOKEN" \
  -d '{
    "name": "Rerank KB",
    "rag_type": "rerank"
  }'
```

## 性能预期

| 部署模式 | 延迟/查询 | 备注 |
|----------|-----------|------|
| GPU (v2-m3) | ~50ms | 推荐生产环境 |
| GPU (v2-minico) | ~15ms | 低延迟场景 |
| CPU (v2-m3) | ~500ms | 开发测试 |

## 下一步

1. **启动 BGE 服务**: `./services/bge-rerank/start.sh`
2. **测试服务**: `./services/bge-rerank/test.sh`
3. **配置 myRAG**: 设置 `BGE_RERANK_ENABLED=true`
4. **验证编译**: `go build ./...`
