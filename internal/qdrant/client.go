package qdrant

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	qdrant "github.com/qdrant/go-client/qdrant"
	"go.uber.org/zap"
)

// =============================================================================
// 重新导出 Qdrant 类型，供其他包使用
// =============================================================================
type (
	Filter               = qdrant.Filter         // 过滤器，用于条件查询
	Condition            = qdrant.Condition      // 查询条件
	FieldCondition       = qdrant.FieldCondition // 字段条件
	Match                = qdrant.Match          // 匹配条件
	MatchKeyword         = qdrant.Match_Keyword  // 关键词匹配
	PointStruct          = qdrant.PointStruct    // 点结构（向量 + 负载）
	UpsertPoints         = qdrant.UpsertPoints   // 插入/更新点
	DeletePoints         = qdrant.DeletePoints   // 删除点
	QueryPoints          = qdrant.QueryPoints    // 查询点
	PointsSelector       = qdrant.PointsSelector // 点选择器
	PointsSelectorFilter = qdrant.PointsSelector_Filter // 过滤器点选择器
	Value                = qdrant.Value          // 负载值
	Query                = qdrant.Query          // 查询输入
	WithPayloadSelector  = qdrant.WithPayloadSelector // 负载选择器
	ValueStringValue     = qdrant.Value_StringValue   // 字符串负载值
	ValueIntegerValue    = qdrant.Value_IntegerValue  // 整数负载值
	PointId              = qdrant.PointId        // 点 ID
)

// =============================================================================
// 辅助函数 - 用于创建 Qdrant 对象
// =============================================================================

// NewConditionField 从字段条件创建查询条件
// 参数:
//   - field: 字段条件指针
// 返回:
//   - *Condition: 包装后的查询条件
func NewConditionField(field *FieldCondition) *Condition {
	return &Condition{
		ConditionOneOf: &qdrant.Condition_Field{
			Field: field,
		},
	}
}

// NewMatchKeyword 创建关键词匹配条件
// 用于精确匹配字段值（不分词）
// 参数:
//   - value: 要匹配的关键词
// 返回:
//   - *Match: 匹配条件指针
func NewMatchKeyword(value string) *Match {
	return &Match{
		MatchValue: &MatchKeyword{
			Keyword: value,
		},
	}
}

// NewQueryWrapper 封装 qdrant.NewQuery 函数
// 创建向量查询输入
// 参数:
//   - vector: 向量数据（float32 数组）
// 返回:
//   - *Query: 查询输入指针
func NewQueryWrapper(vector ...float32) *Query {
	return qdrant.NewQuery(vector...)
}

// NewWithPayloadWrapper 创建负载选择器
// 用于指定查询是否返回负载数据
// 参数:
//   - withPayload: 是否返回负载
// 返回:
//   - *WithPayloadSelector: 负载选择器指针
func NewWithPayloadWrapper(withPayload bool) *WithPayloadSelector {
	return qdrant.NewWithPayload(withPayload)
}

// NewIDUUID 从 UUID 字符串创建点 ID
// 参数:
//   - uuid: UUID 字符串
// 返回:
//   - *PointId: 点 ID 指针
func NewIDUUID(uuid string) *PointId {
	return qdrant.NewIDUUID(uuid)
}

// NewValueString 创建字符串负载值
// 用于构建点结构的负载数据
// 参数:
//   - value: 字符串值
// 返回:
//   - *Value: 负载值指针
func NewValueString(value string) *Value {
	return &Value{
		Kind: &ValueStringValue{
			StringValue: value,
		},
	}
}

// NewValueInteger 创建整数负载值
// 用于构建点结构的负载数据
// 参数:
//   - value: 整数值
// 返回:
//   - *Value: 负载值指针
func NewValueInteger(value int64) *Value {
	return &Value{
		Kind: &ValueIntegerValue{
			IntegerValue: value,
		},
	}
}

// NewVectorsWrapper 封装 qdrant.NewVectors 函数
// 创建向量对象
// 参数:
//   - vector: 向量数据（float32 数组）
// 返回:
//   - *qdrant.Vectors: 向量对象指针
func NewVectorsWrapper(vector ...float32) *qdrant.Vectors {
	return qdrant.NewVectors(vector...)
}

// NewFilter 创建新的过滤器，包含多个 AND 条件
// 参数:
//   - conditions: 条件列表（可变参数）
// 返回:
//   - *Filter: 过滤器指针
func NewFilter(conditions ...*Condition) *Filter {
	return &Filter{
		Must: conditions,
	}
}

// NewPointsSelectorFilter 从过滤器创建点选择器
// 用于删除操作时指定要删除的点
// 参数:
//   - filter: 过滤器指针
// 返回:
//   - *PointsSelector: 点选择器指针
func NewPointsSelectorFilter(filter *Filter) *PointsSelector {
	return qdrant.NewPointsSelectorFilter(filter)
}

// =============================================================================
// Client - Qdrant 客户端包装
// =============================================================================

// Client 封装 Qdrant 客户端，提供应用级辅助方法
// 包含基础的 Qdrant 客户端、日志器和集合名称
type Client struct {
	*qdrant.Client
	log            *zap.Logger  // 日志器
	collectionName string       // 集合名称
}

// New 创建新的 Qdrant 客户端并确保集合存在
// 参数:
//   - url: Qdrant 服务 URL（如 http://localhost:6333）
//   - apiKey: API 密钥
//   - collectionName: 集合名称
//   - log: Zap 日志器
// 返回:
//   - *Client: Qdrant 客户端指针
//   - error: 错误信息
func New(url, apiKey, collectionName string, log *zap.Logger) (*Client, error) {
	// 解析 URL 提取主机和端口
	// Qdrant URL 格式：http://localhost:6333
	// gRPC 需要：localhost:6334
	host, port := extractHostAndPort(url)
	log.Info("Qdrant config", zap.String("input_url", url), zap.String("extracted_host", host), zap.Int("extracted_port", port))

	// 创建 Qdrant 客户端
	client, err := qdrant.NewClient(&qdrant.Config{
		Host: host,
		Port: port,
		APIKey: apiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Qdrant client: %w", err)
	}

	log.Info("Qdrant client connected", zap.String("url", url))

	c := &Client{
		Client:         client,
		log:            log,
		collectionName: collectionName,
	}

	// 确保集合存在
	if err := c.ensureCollection(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ensure collection: %w", err)
	}

	return c, nil
}

// extractHostAndPort 从 URL 中提取主机和端口，用于 gRPC 连接
// Qdrant REST API 使用 6333 端口，gRPC 使用 6334 端口
// 参数:
//   - url: Qdrant URL（如 http://localhost:6333）
// 返回:
//   - string: 主机名
//   - int: 端口号
func extractHostAndPort(url string) (string, int) {
	// 移除 http:// 或 https:// 前缀
	host := strings.TrimPrefix(url, "http://")
	host = strings.TrimPrefix(host, "https://")

	// 默认 gRPC 端口
	port := 6334

	// 处理不同端口场景
	if strings.HasSuffix(host, ":6333") {
		// REST API 端口，切换到 gRPC 端口
		host = strings.TrimSuffix(host, ":6333")
	} else if strings.Contains(host, ":") {
		// 指定了自定义端口，提取它
		parts := strings.SplitN(host, ":", 2)
		if len(parts) == 2 {
			host = parts[0]
			if p, err := strconv.Atoi(parts[1]); err == nil {
				// 如果是 REST 端口，使用 gRPC 端口
				if p == 6333 {
					port = 6334
				} else {
					port = p
				}
			}
		}
	}

	// 确保主机名不包含冒号
	host = strings.Split(host, ":")[0]

	return host, port
}

// ensureCollection 确保文档集合存在，如果不存在则创建
// 使用 qwen3-embedding 模型的向量维度（4096）
// 参数:
//   - ctx: 上下文
// 返回:
//   - error: 错误信息
func (c *Client) ensureCollection(ctx context.Context) error {
	collections, err := c.ListCollections(ctx)
	if err != nil {
		return fmt.Errorf("failed to list collections: %w", err)
	}

	// 检查集合是否已存在（ListCollections 返回 []string）
	for _, colName := range collections {
		if colName == c.collectionName {
			c.log.Info("Qdrant collection exists", zap.String("collection", c.collectionName))
			return nil
		}
	}

	// 创建集合
	// 使用 qwen3-embedding 模型维度（4096）
	vectorSize := uint64(4096)

	err = c.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: c.collectionName,
		VectorsConfig: &qdrant.VectorsConfig{
			Config: &qdrant.VectorsConfig_Params{
				Params: &qdrant.VectorParams{
					Size:     vectorSize,
					Distance: qdrant.Distance_Cosine, // 余弦相似度
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	c.log.Info("Qdrant collection created",
		zap.String("collection", c.collectionName),
		zap.Uint64("vector_size", vectorSize))

	return nil
}

// Close 关闭 Qdrant 客户端
// 目前无需特殊清理操作，保留接口以备将来扩展
func (c *Client) Close() error {
	c.log.Info("closing Qdrant client")
	return nil
}

// GetCollectionName 返回集合名称
// 用于其他包访问当前客户端使用的集合
func (c *Client) GetCollectionName() string {
	return c.collectionName
}

// Chunk 表示要存储到 Qdrant 的文档块
// 包含文档的分片内容、向量嵌入和元数据
type Chunk struct {
	ID         uuid.UUID       `json:"id"`           // 块唯一标识
	DocumentID uuid.UUID       `json:"document_id"`  // 所属文档 ID
	TenantID   uuid.UUID       `json:"tenant_id"`    // 租户 ID（多租户隔离）
	KBID       uuid.UUID       `json:"kb_id"`        // 知识库 ID
	Content    string          `json:"content"`      // 块内容
	ChunkIndex int             `json:"chunk_index"`  // 块索引（在文档中的位置）
	Embedding  []float32       `json:"embedding"`    // 向量嵌入
	Metadata   map[string]any  `json:"metadata,omitempty"` // 额外元数据
}

// UpsertChunks 批量插入/更新文档块到集合
// 参数:
//   - ctx: 上下文
//   - chunks: 文档块切片
// 返回:
//   - error: 错误信息
func (c *Client) UpsertChunks(ctx context.Context, chunks []Chunk) error {
	if len(chunks) == 0 {
		return nil
	}

	points := make([]*qdrant.PointStruct, len(chunks))
	for i, chunk := range chunks {
		// 构建负载数据
		payload := map[string]*qdrant.Value{
			"document_id": {Kind: &qdrant.Value_StringValue{StringValue: chunk.DocumentID.String()}},
			"tenant_id":   {Kind: &qdrant.Value_StringValue{StringValue: chunk.TenantID.String()}},
			"kb_id":       {Kind: &qdrant.Value_StringValue{StringValue: chunk.KBID.String()}},
			"content":     {Kind: &qdrant.Value_StringValue{StringValue: chunk.Content}},
			"chunk_index": {Kind: &qdrant.Value_IntegerValue{IntegerValue: int64(chunk.ChunkIndex)}},
		}

		// 添加可选元数据
		if chunk.Metadata != nil {
			metaJSON, err := json.Marshal(chunk.Metadata)
			if err == nil {
				payload["metadata"] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: string(metaJSON)}}
			}
		}

		pointID := qdrant.NewIDUUID(chunk.ID.String())
		points[i] = &qdrant.PointStruct{
			Id:      pointID,
			Vectors: qdrant.NewVectors(chunk.Embedding...),
			Payload: payload,
		}
	}

	// 等待写入完成
	wait := true
	_, err := c.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: c.collectionName,
		Points:         points,
		Wait:           &wait,
	})

	if err != nil {
		return fmt.Errorf("failed to upsert chunks: %w", err)
	}

	c.log.Debug("upserted chunks",
		zap.String("collection", c.collectionName),
		zap.Int("count", len(chunks)))

	return nil
}

// SearchRequest 表示搜索请求
// 包含查询文本、向量、租户/知识库 ID 和结果限制
type SearchRequest struct {
	Query       string      `json:"query"`                // 查询文本
	QueryVector []float32   `json:"query_vector"`         // 查询向量
	TenantID    uuid.UUID   `json:"tenant_id"`            // 租户 ID
	KBID        uuid.UUID   `json:"kb_id"`                // 知识库 ID
	Limit       int         `json:"limit"`                // 返回结果数量限制
	ScoreThreshold *float32 `json:"score_threshold,omitempty"` // 分数阈值
}

// SearchResult 表示搜索结果
// 包含匹配的文档块信息和相似度分数
type SearchResult struct {
	ID         uuid.UUID       `json:"id"`           // 结果 ID
	DocumentID uuid.UUID       `json:"document_id"`  // 文档 ID
	Content    string          `json:"content"`      // 块内容
	ChunkIndex int             `json:"chunk_index"`  // 块索引
	Score      float32         `json:"score"`        // 相似度分数
	Metadata   map[string]any  `json:"metadata,omitempty"` // 额外元数据
}

// Search 执行相似性搜索，带租户和知识库过滤
// 参数:
//   - ctx: 上下文
//   - req: 搜索请求
// 返回:
//   - []SearchResult: 搜索结果列表
//   - error: 错误信息
func (c *Client) Search(ctx context.Context, req SearchRequest) ([]SearchResult, error) {
	if req.Limit <= 0 {
		req.Limit = 10
	}

	// 构建租户和知识库隔离过滤器
	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "tenant_id",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{
								Keyword: req.TenantID.String(),
							},
						},
					},
				},
			},
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "kb_id",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{
								Keyword: req.KBID.String(),
							},
						},
					},
				},
			},
		},
	}

	// 执行搜索
	points, err := c.Query(ctx, &qdrant.QueryPoints{
		CollectionName: c.collectionName,
		Query:          qdrant.NewQuery(req.QueryVector...),
		Filter:         filter,
		Limit:          &[]uint64{uint64(req.Limit)}[0],
		WithPayload:    qdrant.NewWithPayload(true),
	})

	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	results := make([]SearchResult, len(points))
	for i, point := range points {
		result := SearchResult{
			Score: point.Score,
		}

		// 提取 ID
		if point.Id != nil {
			idStr := point.Id.GetUuid()
			if idStr != "" {
				result.ID, _ = uuid.Parse(idStr)
			}
		}

		// 提取负载数据
		if point.Payload != nil {
			if docID, ok := point.Payload["document_id"]; ok {
				result.DocumentID, _ = uuid.Parse(docID.GetStringValue())
			}
			if content, ok := point.Payload["content"]; ok {
				result.Content = content.GetStringValue()
			}
			if chunkIndex, ok := point.Payload["chunk_index"]; ok {
				result.ChunkIndex = int(chunkIndex.GetIntegerValue())
			}
			if metadata, ok := point.Payload["metadata"]; ok {
				// 如需解析元数据 JSON，可在此处理
				_ = metadata
			}
		}

		results[i] = result
	}

	return results, nil
}

// DeleteByDocumentID 删除文档关联的所有文档块
// 通过租户 ID、知识库 ID 和文档 ID 进行精确过滤
// 参数:
//   - ctx: 上下文
//   - tenantID: 租户 ID
//   - kbID: 知识库 ID
//   - docID: 文档 ID
// 返回:
//   - error: 错误信息
func (c *Client) DeleteByDocumentID(ctx context.Context, tenantID, kbID, docID uuid.UUID) error {
	// 构建过滤器：文档 ID + 租户 ID
	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "document_id",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{
								Keyword: docID.String(),
							},
						},
					},
				},
			},
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "tenant_id",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{
								Keyword: tenantID.String(),
							},
						},
					},
				},
			},
		},
	}

	// 等待删除完成
	wait := true
	_, err := c.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: c.collectionName,
		Points:         qdrant.NewPointsSelectorFilter(filter),
		Wait:           &wait,
	})

	if err != nil {
		return fmt.Errorf("failed to delete chunks: %w", err)
	}

	c.log.Debug("deleted chunks for document",
		zap.String("collection", c.collectionName),
		zap.String("document_id", docID.String()))

	return nil
}

// DeleteByKBID 删除知识库关联的所有文档块
// 通过租户 ID 和知识库 ID 进行过滤
// 参数:
//   - ctx: 上下文
//   - tenantID: 租户 ID
//   - kbID: 知识库 ID
// 返回:
//   - error: 错误信息
func (c *Client) DeleteByKBID(ctx context.Context, tenantID, kbID uuid.UUID) error {
	// 构建过滤器：知识库 ID + 租户 ID
	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "kb_id",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{
								Keyword: kbID.String(),
							},
						},
					},
				},
			},
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "tenant_id",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{
								Keyword: tenantID.String(),
							},
						},
					},
				},
			},
		},
	}

	wait := true
	_, err := c.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: c.collectionName,
		Points:         qdrant.NewPointsSelectorFilter(filter),
		Wait:           &wait,
	})

	if err != nil {
		return fmt.Errorf("failed to delete chunks: %w", err)
	}

	c.log.Debug("deleted chunks for knowledge base",
		zap.String("collection", c.collectionName),
		zap.String("kb_id", kbID.String()))

	return nil
}
