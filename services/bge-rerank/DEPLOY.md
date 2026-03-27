# BGE Rerank 部署指南

## 部署步骤

### 1. 启动 BGE 服务

```bash
cd services/bge-rerank

# 方式 1: 使用启动脚本（推荐）
./start.sh

# 方式 2: 手动启动
pip install -r requirements.txt
python main.py
```

服务启动后运行在 `http://localhost:8800`

### 2. 测试服务

```bash
# 使用测试脚本
./test.sh

# 或手动测试
curl http://localhost:8800/health
```

### 3. 配置 myRAG

编辑 `.env` 文件：

```bash
BGE_RERANK_ENABLED=true
BGE_RERANK_BASE_URL=http://localhost:8800
BGE_RERANK_TOP_K=10
BGE_RERANK_CANDIDATES=50
```

### 4. 启动 myRAG

```bash
go run cmd/server/main.go
```

## 首次启动说明

首次启动时会下载 BGE 模型（约 2GB），建议：

1. 使用镜像加速（国内）：
   ```bash
   export HF_ENDPOINT=https://hf-mirror.com
   ```

2. 或手动下载模型到 `~/.cache/huggingface/hub/`

## 故障排查

| 问题 | 解决方案 |
|------|----------|
| 端口被占用 | 修改 `BGE_PORT` 环境变量 |
| 模型下载慢 | 使用 HF_ENDPOINT 镜像 |
| 内存不足 | 使用 `v2-minico` 模型 |
| GPU 不可用 | 设置 `BGE_DEVICE=cpu` |
