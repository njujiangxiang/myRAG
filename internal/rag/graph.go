package rag

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/google/uuid"

	"myrag/internal/embedding"
	"myrag/internal/qdrant"
)

// 分数权重常量 - 用于不同类型搜索结果的加权
const (
	EntityScoreWeight    = 1.0   // 实体结果权重（最高）
	RelationshipWeight   = 0.8   // 关系结果权重（中等）
	ChunkScoreWeight     = 0.6   // 文档块结果权重（最低）
	MaxContentTruncation = 3000  // 内容截断最大值（LLM 输入限制）
)

// GraphRAG 实现基于知识图谱增强的搜索
// 通过提取文档中的实体和关系，构建图谱结构，实现更深层次的语义搜索
type GraphRAG struct {
	qdrantClient    *qdrant.Client      // Qdrant 向量数据库客户端
	embeddingClient *embedding.Client   // 文本嵌入生成客户端
	llmClient       *LLMClient          // LLM 客户端（用于实体提取）
	graphMutex      sync.RWMutex        // 图谱操作读写锁
}

// Entity 表示知识图谱中的实体
// 实体是从文档中提取的关键信息单元，如人物、组织、地点等
type Entity struct {
	ID          uuid.UUID      `json:"id"`           // 实体唯一标识
	Name        string         `json:"name"`         // 实体名称
	Type        EntityType     `json:"type"`         // 实体类型
	Description string         `json:"description"`  // 实体描述
	DocID       uuid.UUID      `json:"document_id"`  // 来源文档 ID
	KBID        uuid.UUID      `json:"kb_id"`        // 知识库 ID
	TenantID    uuid.UUID      `json:"tenant_id"`    // 租户 ID（多租户隔离）
	Metadata    map[string]any `json:"metadata,omitempty"` // 额外元数据
	Embedding   []float32      `json:"-"`            // 向量嵌入（不序列化）
}

// EntityType 定义实体类型枚举
type EntityType string

const (
	EntityPerson       EntityType = "person"        // 人物
	EntityOrganization EntityType = "organization"  // 组织/机构
	EntityLocation     EntityType = "location"      // 地点
	EntityConcept      EntityType = "concept"       // 概念
	EntityEvent        EntityType = "event"         // 事件
	EntityProduct      EntityType = "product"       // 产品
	EntityTechnology   EntityType = "technology"    // 技术
	EntityUnknown      EntityType = "unknown"       // 未知类型
)

// Relationship 表示实体之间的关系连接
// 关系用于描述两个实体之间的语义关联
type Relationship struct {
	ID          uuid.UUID      `json:"id"`           // 关系唯一标识
	SourceID    uuid.UUID      `json:"source_id"`    // 源实体 ID
	TargetID    uuid.UUID      `json:"target_id"`    // 目标实体 ID
	Type        RelationshipType `json:"type"`       // 关系类型
	Description string         `json:"description"`  // 关系描述
	DocID       uuid.UUID      `json:"document_id"`  // 来源文档 ID
	KBID        uuid.UUID      `json:"kb_id"`        // 知识库 ID
	TenantID    uuid.UUID      `json:"tenant_id"`    // 租户 ID（多租户隔离）
	Metadata    map[string]any `json:"metadata,omitempty"` // 额外元数据
	Embedding   []float32      `json:"-"`            // 向量嵌入（不序列化）
}

// RelationshipType 定义关系类型枚举
type RelationshipType string

const (
	RelWorksFor    RelationshipType = "works_for"     // 任职于
	RelLocatedIn   RelationshipType = "located_in"    // 位于
	RelRelatedTo   RelationshipType = "related_to"    // 相关于
	RelPartOf      RelationshipType = "part_of"       // 属于
	RelCaused      RelationshipType = "caused"        // 导致
	RelUses        RelationshipType = "uses"          // 使用
	RelDevelopedBy RelationshipType = "developed_by"  // 由...开发
	RelDependsOn   RelationshipType = "depends_on"    // 依赖于
	RelSimilarTo   RelationshipType = "similar_to"    // 类似于
	RelUnknown     RelationshipType = "unknown"       // 未知关系
)

// EntityExtractionResult 表示从文本中提取的实体和关系结果
type EntityExtractionResult struct {
	Entities      []Entity        `json:"entities"`       // 提取的实体列表
	Relationships []relationshipRaw `json:"relationships"` // 提取的关系列表（原始格式）
}

// relationshipRaw 用于提取阶段的原始关系结构
// 此时源实体和目标实体仍使用名称（字符串）而非 UUID
type relationshipRaw struct {
	Source string `json:"source"` // 源实体名称
	Target string `json:"target"` // 目标实体名称
	Type   string `json:"type"`   // 关系类型
}

// NewGraphRAG 创建一个新的 Graph RAG 策略实例
// 参数:
//   - qdrantClient: Qdrant 向量数据库客户端
//   - embeddingClient: 文本嵌入生成客户端
//   - llmClient: LLM 客户端（用于实体关系提取）
func NewGraphRAG(qdrantClient *qdrant.Client, embeddingClient *embedding.Client, llmClient *LLMClient) *GraphRAG {
	return &GraphRAG{
		qdrantClient:    qdrantClient,
		embeddingClient: embeddingClient,
		llmClient:       llmClient,
	}
}

// GetType 返回 RAG 策略类型
func (g *GraphRAG) GetType() RAGType {
	return RAGTypeGraph
}

// Search 执行图谱增强搜索
// 搜索流程：1) 生成查询向量 2) 搜索实体 3) 搜索关系 4) 搜索文档块 5) 合并结果
// 参数:
//   - ctx: 上下文
//   - query: 搜索查询文本
//   - kbID: 知识库 ID
//   - tenantID: 租户 ID
//   - limit: 返回结果数量限制
func (g *GraphRAG) Search(ctx context.Context, query string, kbID, tenantID uuid.UUID, limit int) ([]SearchResult, error) {
	// 步骤 1: 为查询生成向量嵌入
	queryVector, err := g.embeddingClient.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// 步骤 2: 搜索匹配的实体
	entityResults, err := g.searchEntities(ctx, query, queryVector, kbID, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("entity search failed: %w", err)
	}

	// 步骤 3: 通过图谱遍历搜索相关实体
	relatedResults, err := g.searchRelatedEntities(ctx, query, queryVector, kbID, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("related entity search failed: %w", err)
	}

	// 步骤 4: 搜索文档块（作为后备）
	chunkResults, err := g.searchChunks(ctx, query, queryVector, kbID, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("chunk search failed: %w", err)
	}

	// 步骤 5: 合并和去重结果
	mergedResults := g.mergeResults(entityResults, relatedResults, chunkResults, limit)

	return mergedResults, nil
}

// searchEntities 搜索与查询匹配的实体
// 使用向量相似度搜索，同时通过租户和知识库 ID 进行数据隔离
func (g *GraphRAG) searchEntities(ctx context.Context, query string, queryVector []float32, kbID, tenantID uuid.UUID, limit int) ([]SearchResult, error) {
	g.graphMutex.RLock()
	defer g.graphMutex.RUnlock()

	// 构建实体搜索过滤器：租户隔离 + 知识库隔离 + 实体类型过滤
	filter := qdrant.NewFilter(
		qdrant.NewConditionField(&qdrant.FieldCondition{
			Key:   "tenant_id",
			Match: qdrant.NewMatchKeyword(tenantID.String()),
		}),
		qdrant.NewConditionField(&qdrant.FieldCondition{
			Key:   "kb_id",
			Match: qdrant.NewMatchKeyword(kbID.String()),
		}),
		qdrant.NewConditionField(&qdrant.FieldCondition{
			Key:   "item_type",
			Match: qdrant.NewMatchKeyword("entity"),
		}),
	)

	limitUint := uint64(limit)
	points, err := g.qdrantClient.Query(ctx, &qdrant.QueryPoints{
		CollectionName: g.qdrantClient.GetCollectionName(),
		Query:          qdrant.NewQueryWrapper(queryVector...),
		Filter:         filter,
		Limit:          &limitUint,
		WithPayload:    qdrant.NewWithPayloadWrapper(true),
	})
	if err != nil {
		return nil, err
	}

	results := make([]SearchResult, len(points))
	for i, point := range points {
		var entity Entity
		if point.Payload != nil {
			// 从向量数据库负载中提取实体信息
			if name, ok := point.Payload["name"]; ok {
				entity.Name = name.GetStringValue()
			}
			if entityType, ok := point.Payload["entity_type"]; ok {
				entity.Type = EntityType(entityType.GetStringValue())
			}
			if description, ok := point.Payload["description"]; ok {
				entity.Description = description.GetStringValue()
			}
			if docID, ok := point.Payload["document_id"]; ok {
				entity.DocID, _ = uuid.Parse(docID.GetStringValue())
			}
		}

		results[i] = SearchResult{
			DocumentID: entity.DocID,
			Content:    fmt.Sprintf("[Entity: %s] %s (Type: %s)", entity.Name, entity.Description, entity.Type),
			Score:      point.Score,
			Metadata: map[string]any{
				"entity_id":   point.Id.GetUuid(),
				"entity_name": entity.Name,
				"entity_type": string(entity.Type),
				"result_type": "entity",
			},
		}
	}

	return results, nil
}

// searchRelatedEntities 搜索与顶层匹配实体相关的其他实体
// 通过关系图谱进行遍历，发现间接相关的信息
func (g *GraphRAG) searchRelatedEntities(ctx context.Context, query string, queryVector []float32, kbID, tenantID uuid.UUID, limit int) ([]SearchResult, error) {
	g.graphMutex.RLock()
	defer g.graphMutex.RUnlock()

	// 构建关系搜索过滤器：租户隔离 + 知识库隔离 + 关系类型过滤
	filter := qdrant.NewFilter(
		qdrant.NewConditionField(&qdrant.FieldCondition{
			Key:   "tenant_id",
			Match: qdrant.NewMatchKeyword(tenantID.String()),
		}),
		qdrant.NewConditionField(&qdrant.FieldCondition{
			Key:   "kb_id",
			Match: qdrant.NewMatchKeyword(kbID.String()),
		}),
		qdrant.NewConditionField(&qdrant.FieldCondition{
			Key:   "item_type",
			Match: qdrant.NewMatchKeyword("relationship"),
		}),
	)

	limitUint := uint64(limit)
	points, err := g.qdrantClient.Query(ctx, &qdrant.QueryPoints{
		CollectionName: g.qdrantClient.GetCollectionName(),
		Query:          qdrant.NewQueryWrapper(queryVector...),
		Filter:         filter,
		Limit:          &limitUint,
		WithPayload:    qdrant.NewWithPayloadWrapper(true),
	})
	if err != nil {
		return nil, err
	}

	results := make([]SearchResult, len(points))
	for i, point := range points {
		var rel Relationship
		if point.Payload != nil {
			// 从向量数据库负载中提取关系信息
			if relType, ok := point.Payload["relationship_type"]; ok {
				rel.Type = RelationshipType(relType.GetStringValue())
			}
			if description, ok := point.Payload["description"]; ok {
				rel.Description = description.GetStringValue()
			}
			if sourceID, ok := point.Payload["source_id"]; ok {
				rel.SourceID, _ = uuid.Parse(sourceID.GetStringValue())
			}
			if targetID, ok := point.Payload["target_id"]; ok {
				rel.TargetID, _ = uuid.Parse(targetID.GetStringValue())
			}
			if docID, ok := point.Payload["document_id"]; ok {
				rel.DocID, _ = uuid.Parse(docID.GetStringValue())
			}
		}

		results[i] = SearchResult{
			DocumentID: rel.DocID,
			Content:    fmt.Sprintf("[Relationship: %s] %s -> %s", rel.Type, rel.SourceID.String()[:8], rel.TargetID.String()[:8]),
			Score:      point.Score * RelationshipWeight, // 关系结果权重略低
			Metadata: map[string]any{
				"relationship_id":   point.Id.GetUuid(),
				"relationship_type": string(rel.Type),
				"source_id":         rel.SourceID.String(),
				"target_id":         rel.TargetID.String(),
				"result_type":       "relationship",
			},
		}
	}

	return results, nil
}

// searchChunks 搜索文档块（作为后备机制）
// 当实体和关系搜索没有足够结果时，回退到传统向量搜索
func (g *GraphRAG) searchChunks(ctx context.Context, query string, queryVector []float32, kbID, tenantID uuid.UUID, limit int) ([]SearchResult, error) {
	results, err := g.qdrantClient.Search(ctx, qdrant.SearchRequest{
		QueryVector: queryVector,
		TenantID:    tenantID,
		KBID:        kbID,
		Limit:       limit,
	})
	if err != nil {
		return nil, err
	}

	searchResults := make([]SearchResult, len(results))
	for i, r := range results {
		searchResults[i] = SearchResult{
			DocumentID: r.DocumentID,
			Content:    r.Content,
			Score:      r.Score * ChunkScoreWeight, // 文档块权重最低
			Metadata: map[string]any{
				"result_type": "chunk",
				"chunk_index": r.ChunkIndex,
			},
		}
	}

	return searchResults, nil
}

// mergeResults 合并和去重搜索结果
// 合并来自实体、关系和文档块的结果，按分数排序并去重
func (g *GraphRAG) mergeResults(entityResults, relatedResults, chunkResults []SearchResult, limit int) []SearchResult {
	// 合并所有结果
	allResults := append(append(entityResults, relatedResults...), chunkResults...)

	// 按分数降序排序
	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].Score > allResults[j].Score
	})

	// 基于文档 ID 和内容哈希进行去重
	seen := make(map[string]bool)
	unique := make([]SearchResult, 0, limit)

	for _, r := range allResults {
		// 使用 SHA256 哈希确保可靠的去重
		hash := sha256.Sum256([]byte(r.Content))
		key := fmt.Sprintf("%s_%x", r.DocumentID.String(), hash[:8])
		if !seen[key] {
			seen[key] = true
			unique = append(unique, r)
		}
		if len(unique) >= limit {
			break
		}
	}

	return unique
}

// IndexDocument 处理文档并构建图谱结构
// 将文档中的实体和关系提取出来，生成向量嵌入，存储到 Qdrant
// 参数:
//   - ctx: 上下文
//   - docID: 文档 ID
//   - kbID: 知识库 ID
//   - tenantID: 租户 ID
//   - content: 文档内容
//   - mimeType: 文档 MIME 类型（暂未使用）
func (g *GraphRAG) IndexDocument(ctx context.Context, docID, kbID, tenantID uuid.UUID, content string, mimeType string) error {
	g.graphMutex.Lock()
	defer g.graphMutex.Unlock()

	// 步骤 1: 使用 LLM 提取实体和关系
	extractionResult, err := g.extractEntities(ctx, content, docID, kbID, tenantID)
	if err != nil {
		return fmt.Errorf("entity extraction failed: %w", err)
	}

	// 步骤 2: 为实体生成向量嵌入
	for i := range extractionResult.Entities {
		embedding, err := g.embeddingClient.GenerateEmbedding(ctx, extractionResult.Entities[i].Name+" "+extractionResult.Entities[i].Description)
		if err != nil {
			return fmt.Errorf("failed to generate embedding for entity: %w", err)
		}
		extractionResult.Entities[i].Embedding = embedding
	}

	// 步骤 3: 为关系生成向量嵌入并转换为正式结构
	relationships := make([]Relationship, 0, len(extractionResult.Relationships))

	// 建立实体名称到 ID 的映射，用于关系解析
	entityNameToID := make(map[string]uuid.UUID)
	for _, e := range extractionResult.Entities {
		entityNameToID[strings.ToLower(e.Name)] = e.ID
	}

	for _, rel := range extractionResult.Relationships {
		// 生成关系描述和嵌入
		relDesc := fmt.Sprintf("%s connects %s to %s", rel.Type, rel.Source, rel.Target)
		embedding, err := g.embeddingClient.GenerateEmbedding(ctx, relDesc)
		if err != nil {
			return fmt.Errorf("failed to generate embedding for relationship: %w", err)
		}

		// 将关系中的源实体和目标实体名称解析为 UUID
		sourceID, ok1 := entityNameToID[strings.ToLower(rel.Source)]
		targetID, ok2 := entityNameToID[strings.ToLower(rel.Target)]

		// 仅当两个实体都存在时才添加关系
		if ok1 && ok2 {
			relationships = append(relationships, Relationship{
				ID:          uuid.New(),
				SourceID:    sourceID,
				TargetID:    targetID,
				Type:        RelationshipType(rel.Type),
				Description: relDesc,
				DocID:       docID,
				KBID:        kbID,
				TenantID:    tenantID,
				Embedding:   embedding,
			})
		}
	}

	// 步骤 4: 将实体索引到 Qdrant
	if err := g.indexEntities(ctx, extractionResult.Entities); err != nil {
		return fmt.Errorf("failed to index entities: %w", err)
	}

	// 步骤 5: 将关系索引到 Qdrant
	if err := g.indexRelationships(ctx, relationships); err != nil {
		return fmt.Errorf("failed to index relationships: %w", err)
	}

	return nil
}

// extractEntities 使用 LLM 从文本中提取实体和关系
// 通过提示工程让 LLM 以 JSON 格式输出结构化的实体和关系信息
func (g *GraphRAG) extractEntities(ctx context.Context, content string, docID, kbID, tenantID uuid.UUID) (*EntityExtractionResult, error) {
	if g.llmClient == nil {
		// 降级处理：如果 LLM 客户端不可用，返回空结果
		return &EntityExtractionResult{
			Entities:      []Entity{},
			Relationships: []relationshipRaw{},
		}, nil
	}

	// 构建提取提示
	// 如果内容过长则截断
	extractContent := content
	if len(content) > MaxContentTruncation {
		extractContent = content[:MaxContentTruncation]
	}

	prompt := fmt.Sprintf(`Extract all entities and their relationships from the following text.

Return the result in JSON format with this structure:
{
  "entities": [
    {"name": "Entity Name", "type": "person|organization|location|concept|event|product|technology", "description": "Brief description"}
  ],
  "relationships": [
    {"source": "Source Entity Name", "target": "Target Entity Name", "type": "works_for|located_in|related_to|part_of|caused|uses|developed_by|depends_on|similar_to"}
  ]
}

Text to analyze:
%s

Only return the JSON, no other text.`, extractContent)

	// 调用 LLM 进行实体提取
	response, err := g.llmClient.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	// 解析 JSON 响应
	var result EntityExtractionResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	// 为实体添加 ID 和元数据
	for i := range result.Entities {
		result.Entities[i].ID = uuid.New()
		result.Entities[i].DocID = docID
		result.Entities[i].KBID = kbID
		result.Entities[i].TenantID = tenantID
	}

	// 关系中的实体名称到 ID 的映射在 indexRelationships 中处理
	// 原始关系中的 source/target 仍为字符串形式

	return &result, nil
}

// indexEntities 将实体索引到 Qdrant 向量数据库
// 将实体转换为 Qdrant 点结构，包含向量嵌入和负载数据
func (g *GraphRAG) indexEntities(ctx context.Context, entities []Entity) error {
	if len(entities) == 0 {
		return nil
	}

	points := make([]*qdrant.PointStruct, len(entities))
	for i, entity := range entities {
		// 构建负载数据，用于过滤和展示
		payload := map[string]*qdrant.Value{
			"item_type":     qdrant.NewValueString("entity"),
			"name":          qdrant.NewValueString(entity.Name),
			"entity_type":   qdrant.NewValueString(string(entity.Type)),
			"description":   qdrant.NewValueString(entity.Description),
			"document_id":   qdrant.NewValueString(entity.DocID.String()),
			"kb_id":         qdrant.NewValueString(entity.KBID.String()),
			"tenant_id":     qdrant.NewValueString(entity.TenantID.String()),
		}

		pointID := qdrant.NewIDUUID(entity.ID.String())
		points[i] = &qdrant.PointStruct{
			Id:      pointID,
			Vectors: qdrant.NewVectorsWrapper(entity.Embedding...),
			Payload: payload,
		}
	}

	// 等待写入完成
	wait := true
	_, err := g.qdrantClient.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: g.qdrantClient.GetCollectionName(),
		Points:         points,
		Wait:           &wait,
	})

	return err
}

// indexRelationships 将关系索引到 Qdrant 向量数据库
// 将关系转换为 Qdrant 点结构，包含源实体和目标实体的引用
func (g *GraphRAG) indexRelationships(ctx context.Context, relationships []Relationship) error {
	if len(relationships) == 0 {
		return nil
	}

	points := make([]*qdrant.PointStruct, len(relationships))
	for i, rel := range relationships {
		// 构建负载数据，记录关系类型和连接的实体
		payload := map[string]*qdrant.Value{
			"item_type":         qdrant.NewValueString("relationship"),
			"relationship_type": qdrant.NewValueString(string(rel.Type)),
			"description":       qdrant.NewValueString(rel.Description),
			"source_id":         qdrant.NewValueString(rel.SourceID.String()),
			"target_id":         qdrant.NewValueString(rel.TargetID.String()),
			"document_id":       qdrant.NewValueString(rel.DocID.String()),
			"kb_id":             qdrant.NewValueString(rel.KBID.String()),
			"tenant_id":         qdrant.NewValueString(rel.TenantID.String()),
		}

		pointID := qdrant.NewIDUUID(rel.ID.String())
		points[i] = &qdrant.PointStruct{
			Id:      pointID,
			Vectors: qdrant.NewVectorsWrapper(rel.Embedding...),
			Payload: payload,
		}
	}

	// 等待写入完成
	wait := true
	_, err := g.qdrantClient.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: g.qdrantClient.GetCollectionName(),
		Points:         points,
		Wait:           &wait,
	})

	return err
}

// DeleteDocument 从索引中删除文档及其图谱实体
// 删除该文档关联的所有实体、关系和文档块
func (g *GraphRAG) DeleteDocument(ctx context.Context, docID, kbID, tenantID uuid.UUID) error {
	g.graphMutex.Lock()
	defer g.graphMutex.Unlock()

	// 删除实体
	if err := g.deleteDocumentItems(ctx, tenantID, kbID, docID, "entity"); err != nil {
		return fmt.Errorf("failed to delete entities: %w", err)
	}

	// 删除关系
	if err := g.deleteDocumentItems(ctx, tenantID, kbID, docID, "relationship"); err != nil {
		return fmt.Errorf("failed to delete relationships: %w", err)
	}

	// 删除文档块
	if err := g.qdrantClient.DeleteByDocumentID(ctx, tenantID, kbID, docID); err != nil {
		return err
	}

	return nil
}

// deleteDocumentItems 删除文档关联的实体或关系
// 通过文档 ID 和项目类型（实体/关系）构建过滤器进行删除
func (g *GraphRAG) deleteDocumentItems(ctx context.Context, tenantID, kbID, docID uuid.UUID, itemType string) error {
	filter := qdrant.NewFilter(
		qdrant.NewConditionField(&qdrant.FieldCondition{
			Key:   "document_id",
			Match: qdrant.NewMatchKeyword(docID.String()),
		}),
		qdrant.NewConditionField(&qdrant.FieldCondition{
			Key:   "item_type",
			Match: qdrant.NewMatchKeyword(itemType),
		}),
	)

	wait := true
	_, err := g.qdrantClient.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: g.qdrantClient.GetCollectionName(),
		Points:         qdrant.NewPointsSelectorFilter(filter),
		Wait:           &wait,
	})

	return err
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
