# myRAG 启动指南

## 系统要求

- Docker 和 Docker Compose
- Go 1.21+（用于本地开发）
- OpenAI API 密钥（或其他兼容的 LLM API）

## 快速启动（Docker）

### 1. 克隆仓库

```bash
git clone <your-repo-url>
cd myrag
```

### 2. 配置环境变量

```bash
cp .env.example .env
```

编辑 `.env` 文件，设置必要的配置：

```env
# 必填：OpenAI API 配置
OPENAI_API_KEY=sk-your-api-key-here
OPENAI_BASE_URL=https://api.openai.com/v1
EMBEDDING_MODEL=text-embedding-3-small
LLM_MODEL=gpt-4o-mini

# 必填：JWT 密钥（生产环境请修改）
JWT_SECRET=change-this-to-a-random-secret-key-in-production
```

### 3. 启动所有服务

```bash
docker-compose up -d
```

### 4. 检查服务状态

```bash
docker-compose ps
```

所有服务应该显示 `Up` 状态：
- `myrag-app` - 后端 API (端口 8080)
- `myrag-postgres` - PostgreSQL 数据库 (端口 5432)
- `myrag-qdrant` - Qdrant 向量数据库 (端口 6333)
- `myrag-nats` - NATS 消息队列 (端口 4222)
- `myrag-minio` - MinIO 对象存储 (端口 9000, Web 控制台 9001)

### 5. 测试 API

```bash
# 健康检查
curl http://localhost:8080/health

# 应该返回：{"status":"healthy","timestamp":"..."}
```

## 本地开发

### 1. 安装依赖

```bash
go mod download
```

### 2. 启动依赖服务

```bash
docker-compose up -d postgres qdrant nats minio
```

### 3. 配置环境变量

确保 `.env` 文件存在并配置正确。

### 4. 运行后端服务

```bash
go run cmd/server/main.go
```

服务将在 `http://localhost:8080` 启动。

## API 使用指南

### 1. 注册用户

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123",
    "name": "Test User"
  }'
```

### 2. 登录获取 Token

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'
```

保存返回的 `token` 用于后续请求。

### 3. 创建知识库

```bash
curl -X POST http://localhost:8080/api/v1/kbs \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "name": "My Knowledge Base",
    "description": "My first knowledge base"
  }'
```

保存返回的 `kb_id`。

### 4. 上传文档

```bash
curl -X POST http://localhost:8080/api/v1/kbs/YOUR_KB_ID/docs \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -F "file=@/path/to/your/document.pdf"
```

文档将在后台处理，处理完成后状态变为 `indexed`。

### 5. 与知识库聊天

```bash
curl -X POST http://localhost:8080/api/v1/kbs/YOUR_KB_ID/chat \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "content": "文档中提到了什么内容？"
  }'
```

### 6. 搜索文档

```bash
curl -G "http://localhost:8080/api/v1/kbs/YOUR_KB_ID/search" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  --data-urlencode "query=你的搜索关键词" \
  --data-urlencode "limit=10"
```

## 架构说明

### 数据处理流程

1. **上传文档** → MinIO 存储原始文件
2. **发布事件** → NATS JetStream 队列
3. **Worker 处理** → 解析文档 → 分块 → 生成 Embedding → 存储到 Qdrant
4. **状态更新** → PostgreSQL 更新文档状态为 `indexed`

### 多租户隔离

- **PostgreSQL**: Row Level Security (RLS) 基于 `tenant_id`
- **Qdrant**: Payload 过滤，查询时必须提供 `tenant_id` 和 `kb_id`
- **MinIO**: 文件路径包含 `tenant_id` 和 `kb_id` 前缀

### 检索流程

1. 用户提问 → 生成问题 Embedding
2. Qdrant 向量搜索 → 返回最相关的文档块
3. 构建上下文 → 发送給 LLM
4. LLM 生成答案 → 返回给用户

## 故障排除

### 无法启动服务

检查端口是否被占用：
```bash
lsof -i :8080
lsof -i :5432
lsof -i :6333
```

### 文档处理失败

查看 Worker 日志：
```bash
docker-compose logs app
```

### 向量搜索返回空结果

确认文档已完成处理：
```bash
curl http://localhost:8080/api/v1/docs/DOC_ID \
  -H "Authorization: Bearer YOUR_TOKEN"
```
检查 `status` 字段应为 `indexed`。

## 性能调优

### Embedding 批处理

默认批处理大小为 100，可在 `internal/embedding/client.go` 中调整 `batchSize`。

### Qdrant 配置

对于大规模数据，调整 Qdrant 配置：
```yaml
qdrant:
  environment:
    - QDRANT__STORAGE__PERFORMANCE_MAX_OPTIMIZATION_THREADS=8
```

### 并发 Worker

当前为单 Worker 模式，可通过增加 Worker 实例提升处理能力。

## 备份与恢复

### 备份数据

```bash
# PostgreSQL 备份
docker exec myrag-postgres pg_dump -U postgres ragdb > backup.sql

# MinIO 备份
# 使用 mc 工具备份存储桶
mc cp -r myrag/documents ./backup/documents
```

### 恢复数据

```bash
# PostgreSQL 恢复
docker exec -i myrag-postgres psql -U postgres ragdb < backup.sql
```

## 安全建议

1. **生产环境必须修改 JWT_SECRET**
2. **启用 HTTPS** - 使用反向代理（如 nginx）
3. **限制 API 访问** - 配置防火墙规则
4. **定期备份数据** - 设置自动备份任务
5. **监控日志** - 集中日志管理

## 技术支持

如有问题，请提交 Issue 或联系开发团队。
