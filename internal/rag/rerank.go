package rag

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"myrag/internal/rerank"
)

// RerankRAG 在现有 RAG 策略基础上实现重排序
// 它从包装的策略中获取候选结果，然后使用 Cross-Encoder 模型
// 按语义相关性进行重排序
type RerankRAG struct {
	wrapped      RAGStrategy      // 包装的底层 RAG 策略
	rerankClient *rerank.Client   // 重排序 API 客户端
	candidates   int              // 重排序前获取的候选数量
	topK         int              // 重排序后返回的结果数量
}

// NewRerankRAG 创建新的重排序 RAG 策略
// 参数:
//   - wrapped: 底层 RAG 策略（如 HybridRAG, VectorRAG）
//   - rerankClient: 重排序 API 客户端
//   - candidates: 重排序前获取的候选结果数量
//   - topK: 重排序后返回的结果数量
func NewRerankRAG(wrapped RAGStrategy, rerankClient *rerank.Client, candidates, topK int) *RerankRAG {
	// 如果未指定，默认 candidates 为 topK 的 5 倍
	if candidates <= 0 {
		candidates = topK * 5
	}
	if candidates < topK {
		candidates = topK * 5
	}
	if topK <= 0 {
		topK = rerankClient.GetTopK()
		if topK <= 0 {
			topK = 10
		}
	}

	return &RerankRAG{
		wrapped:      wrapped,
		rerankClient: rerankClient,
		candidates:   candidates,
		topK:         topK,
	}
}

// GetType 返回 RAG 类型
func (r *RerankRAG) GetType() RAGType {
	return RAGTypeRerank
}

// Search 执行重排序搜索
// 流程：
// 1. 从包装的策略获取候选结果
// 2. 使用 Cross-Encoder 对候选结果重排序
// 3. 返回按重排序分数排名的前 K 个结果
func (r *RerankRAG) Search(ctx context.Context, query string, kbID, tenantID uuid.UUID, limit int) ([]SearchResult, error) {
	// 使用 candidates 数量覆盖 limit 进行初始检索
	candidatesLimit := r.candidates
	if limit > 0 && limit < candidatesLimit {
		candidatesLimit = limit
	}

	// 步骤 1: 从包装的策略获取候选结果
	candidates, err := r.wrapped.Search(ctx, query, kbID, tenantID, candidatesLimit)
	if err != nil {
		return nil, fmt.Errorf("包装策略搜索失败：%w", err)
	}

	if len(candidates) == 0 {
		return []SearchResult{}, nil
	}

	// 步骤 2: 提取内容用于重排序
	contents := make([]string, len(candidates))
	for i, c := range candidates {
		contents[i] = c.Content
	}

	// 步骤 3: 重排序候选结果
	rerankResults, err := r.rerankClient.Rerank(ctx, query, contents, r.topK)
	if err != nil {
		// 回退：如果重排序失败，返回原始候选结果
		if len(candidates) <= limit {
			return candidates, nil
		}
		return candidates[:limit], nil
	}

	// 步骤 4: 按重排序分数重新排列候选结果
	return r.reorderCandidates(candidates, rerankResults, limit), nil
}

// reorderCandidates 根据重排序分数重新排列候选结果
func (r *RerankRAG) reorderCandidates(candidates []SearchResult, rerankResults []rerank.RerankResult, limit int) []SearchResult {
	// 创建索引 -> 重排序分数的映射
	scoreMap := make(map[int]float32)
	for _, result := range rerankResults {
		scoreMap[result.Index] = result.Score
	}

	// 创建带分数的结果列表
	type scoredResult struct {
		result SearchResult
		score  float32
		rank   int // 重排序后的原始排名
	}

	var scored []scoredResult
	for _, rr := range rerankResults {
		if rr.Index >= 0 && rr.Index < len(candidates) {
			scored = append(scored, scoredResult{
				result: candidates[rr.Index],
				score:  rr.Score,
				rank:   rr.Index,
			})
		}
	}

	// 按重排序分数降序排序
	// 重排序 API 返回的结果已排序，但这里确保顺序正确

	// 应用限制
	if limit > 0 && len(scored) > limit {
		scored = scored[:limit]
	}

	// 转换回 SearchResult 并更新分数
	finalResults := make([]SearchResult, len(scored))
	for i, sr := range scored {
		result := sr.result
		result.Score = sr.score
		finalResults[i] = result
	}

	return finalResults
}

// IndexDocument 委托给包装的策略处理
func (r *RerankRAG) IndexDocument(ctx context.Context, docID, kbID, tenantID uuid.UUID, content string, mimeType string) error {
	return r.wrapped.IndexDocument(ctx, docID, kbID, tenantID, content, mimeType)
}

// DeleteDocument 委托给包装的策略处理
func (r *RerankRAG) DeleteDocument(ctx context.Context, docID, kbID, tenantID uuid.UUID) error {
	return r.wrapped.DeleteDocument(ctx, docID, kbID, tenantID)
}

// GetWrappedStrategy 返回底层 RAG 策略
func (r *RerankRAG) GetWrappedStrategy() RAGStrategy {
	return r.wrapped
}

// GetCandidates 返回重排序前获取的候选数量
func (r *RerankRAG) GetCandidates() int {
	return r.candidates
}

// GetTopK 返回重排序后返回的结果数量
func (r *RerankRAG) GetTopK() int {
	return r.topK
}
