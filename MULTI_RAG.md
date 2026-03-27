# 多 RAG 类型支持 - 实现文档

## 概述

系统现在支持多种 RAG（Retrieval-Augmented Generation）类型，用户可以在创建知识库时选择不同的检索策略。

## 支持的 RAG 类型

### 1. Vector（向量检索）✅ 完整实现

- **类型标识**: `vector`
- **实现状态**: ✅ 完整实现
- **描述**: 使用向量相似度进行语义检索
- **存储**: Qdrant
- **核心库**: `github.com/qdrant/go-client`
- **适用场景**:
  - 语义搜索
  - 概念匹配
  - 模糊查询

### 2. Graph（知识图谱）- 实验性

- **类型标识**: `graph`
- **实现状态**: 🚧 部分实现（v1.0 使用向量检索作为 fallback）
- **描述**: 基于知识图谱的增强检索
- **存储**: Qdrant（向量）
- **未来功能** (v1.1):
  - 实体提取和关系构建
  - 图谱存储（计划：Neo4j）
  - 图谱遍历检索

### 3. Hybrid（混合检索）✅ 已实现

- **类型标识**: `hybrid`
- **实现状态**: ✅ 完整实现
- **描述**: 结合向量相似度和 BM25 关键词匹配
- **存储方案**:
  - 向量部分：Qdrant
  - 关键词部分：Bleve（BM25 算法）
- **检索融合**: RRF (Reciprocal Rank Fusion)
- **索引路径**: `./data/rag-index`
- **核心库**:
  - `github.com/blevesearch/bleve/v2` (BM25)
  - `github.com/qdrant/go-client` (向量)
- **适用场景**:
  - 需要精确匹配的场景
  - 术语搜索 + 语义搜索结合
  - 专业领域问答

### 4. Keyword（关键词检索）⏳ 计划中

- **类型标识**: `keyword`
- **实现状态**: ⏳ 计划中
- **描述**: 使用 BM25 算法进行纯关键词检索
- **计划存储**: Bleve
- **适用场景**:
  - 精确匹配
  - 术语搜索

## 架构设计

### 策略模式

```go
// RAGStrategy 定义 RAG 检索接口
type RAGStrategy interface {
    GetType() RAGType
    Search(ctx context.Context, query string, kbID, tenantID uuid.UUID, limit int) ([]SearchResult, error)
    IndexDocument(ctx context.Context, docID, kbID, tenantID uuid.UUID, content string, mimeType string) error
    DeleteDocument(ctx context.Context, docID, kbID, tenantID uuid.UUID) error
}
```

### 具体实现

| 策略 | 文件 | 状态 |
|------|------|------|
| `VectorRAG` | `internal/rag/vector.go` | ✅ |
| `GraphRAG` | `internal/rag/graph.go` | 🚧 |
| `HybridRAG` | `internal/rag/hybrid.go` | ✅ |

### Factory

```go
// Factory 管理 RAG 策略实例
type Factory struct {
    strategies map[RAGType]RAGStrategy
}

// NewFactory 初始化所有策略
func NewFactory(qdrantClient *qdrant.Client, embeddingClient *embedding.Client, indexPath string) *Factory

// GetStrategyByKB 根据知识库类型返回相应策略
func (f *Factory) GetStrategyByKB(ctx context.Context, kbID, tenantID uuid.UUID, kbRepo KBRepository) (RAGStrategy, error)
```

## 数据库修改

### 迁移文件：002_add_rag_type.sql

```sql
ALTER TABLE knowledge_bases
ADD COLUMN IF NOT EXISTS rag_type VARCHAR(50) DEFAULT 'vector'
CHECK (rag_type IN ('vector', 'graph', 'hybrid', 'keyword'));

CREATE INDEX IF NOT EXISTS idx_kbs_rag_type ON knowledge_bases(rag_type);
```

### 数据模型

```go
type KnowledgeBase struct {
    ID          uuid.UUID  `json:"id"`
    TenantID    uuid.UUID  `json:"tenant_id"`
    OwnerID     uuid.UUID  `json:"owner_id"`
    Name        string     `json:"name"`
    Description *string    `json:"description,omitempty"`
    RAGType     string     `json:"rag_type"`  // 新增字段
    CreatedAt   time.Time  `json:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at"`
}
```

## API 变更

### 创建知识库

**请求**:
```http
POST /api/v1/kbs
Content-Type: application/json
Authorization: Bearer {token}

{
  "name": "我的知识库",
  "description": "描述信息",
  "rag_type": "hybrid"  // 可选：vector, graph, hybrid, keyword
}
```

**响应**:
```json
{
  "id": "uuid",
  "tenant_id": "uuid",
  "owner_id": "uuid",
  "name": "我的知识库",
  "description": "描述信息",
  "rag_type": "hybrid",
  "created_at": "2026-03-26T10:00:00Z",
  "updated_at": "2026-03-26T10:00:00Z"
}
```

### 聊天接口（自动选择 RAG 策略）

**请求**:
```http
POST /api/v1/kbs/:id/chat
Content-Type: application/json
Authorization: Bearer {token}

{
  "content": "问题内容"
}
```

系统会自动根据知识库的 `rag_type` 选择相应的检索策略。

## 前端使用

### 创建知识库

在知识库列表页面，点击"创建知识库"按钮：

1. 输入知识库名称和描述
2. 从下拉菜单选择 RAG 类型
3. 点击"创建知识库"

### RAG 类型显示

知识库卡片上会显示 RAG 类型标签：
- 向量检索（蓝色）
- 知识图谱（紫色）
- 混合检索（绿色）
- 关键词检索（橙色）

## Hybrid RAG 详解

### RRF 融合算法

```go
// RRF 公式：score = sum(1 / (k + rank))
// 关键词结果权重 x2
const rrfK = 60

func rrfFusion(vectorResults, keywordResults []SearchResult, limit int) []SearchResult {
    // 向量检索排名得分：1/(60+rank)
    // 关键词排名得分：2/(60+rank)
    // 按总分排序返回
}
```

### BM25 索引

```go
// Bleve 索引配置
indexMapping := bleve.NewIndexMapping()
indexMapping.TypeField = "type"
indexMapping.DefaultAnalyzer = "en"

// 使用 Scorch 存储引擎（支持 BM25）
index, err = bleve.NewUsing(indexPath, indexMapping, scorch.Name, scorch.Name, nil)
```

### 索引路径

- **默认路径**: `./data/rag-index`
- **存储内容**: BM25 倒排索引
- **持久化**: 索引会持久保存到磁盘

## 部署指南

### 1. 运行数据库迁移

```bash
# 确保数据库连接正常
psql -U postgres -d myrag -f migrations/002_add_rag_type.sql
```

### 2. 重新编译后端

```bash
go build ./cmd/server
```

### 3. 重新构建前端

```bash
cd web
npm run build
```

### 4. 重启服务

```bash
./server
```

### 5. 确保索引目录存在

```bash
mkdir -p ./data/rag-index
```

## 开发路线图

### v1.0 (当前版本)
- ✅ 策略模式架构设计
- ✅ Vector RAG 完整实现
- ✅ Hybrid RAG BM25 实现
- ✅ 数据库模式更新
- ✅ 前端 UI 支持
- ✅ Factory 模式实现

### v1.1 (计划中)
- [ ] Graph RAG 实体提取
- [ ] 知识图谱存储（Neo4j）
- [ ] 图谱遍历算法
- [ ] RRF 权重可配置化

### v1.2 (未来)
- [ ] Keyword RAG 独立实现
- [ ] 更多检索策略（如：元数据过滤、时间衰减）
- [ ] RAG 类型动态切换
- [ ] 检索效果评估工具

## 文件清单

### 新增文件
- `internal/rag/strategy.go` - RAG 策略接口定义
- `internal/rag/vector.go` - Vector RAG 实现
- `internal/rag/graph.go` - Graph RAG 实现
- `internal/rag/hybrid.go` - Hybrid RAG 实现（含 BM25）
- `internal/rag/factory.go` - RAG 工厂
- `migrations/002_add_rag_type.sql` - 数据库迁移

### 修改文件
- `internal/models/models.go` - 添加 RAGType 字段
- `internal/models/repository.go` - 更新 Repository 方法
- `internal/handler/kb.go` - 支持 RAG 类型创建和更新
- `internal/handler/chat.go` - 集成 RAG 策略选择
- `cmd/server/main.go` - 初始化 RAG Factory
- `web/src/pages/KBListPage.tsx` - 添加 RAG 类型选择 UI
- `go.mod` - 添加 Bleve 依赖

## 技术栈

### 后端
- Go 1.21+

### 前端
- React + TypeScript

### 向量数据库
- Qdrant 1.7+
- 库：`github.com/qdrant/go-client`

### 全文检索
- Bleve v2
- 存储引擎：Scorch
- BM25 相似度

### 关系数据库
- PostgreSQL 16

### 设计模式
- Strategy Pattern + Factory Pattern

## 性能建议

### Hybrid RAG

1. **索引大小**: BM25 索引会占用额外磁盘空间，建议定期清理
2. **并发安全**: 使用 `sync.RWMutex` 保护索引访问
3. **批处理**: 文档索引时批量 Upsert 到 Qdrant

### Vector RAG

1. **向量维度**: 使用 4096 维（qwen3-embedding）
2. **相似度**: 余弦相似度
3. **批处理**: 支持批量 Upsert

## 常见问题

### Q: Hybrid 和 Vector 有什么区别？

A: Hybrid 同时使用向量相似度和 BM25 关键词匹配，通过 RRF 融合结果。Vector 只使用向量相似度。Hybrid 在精确匹配场景表现更好。

### Q: 如何切换 RAG 类型？

A: RAG 类型在创建知识库时指定，之后不能修改。如需切换，需要创建新的知识库。

### Q: BM25 索引存储在哪里？

A: 默认存储在 `./data/rag-index` 目录。
