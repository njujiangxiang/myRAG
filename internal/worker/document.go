package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go/jetstream"

	"myrag/internal/embedding"
	"myrag/internal/minio"
	"myrag/internal/models"
	"myrag/internal/parser"
	"myrag/internal/qdrant"
)

// DocumentEvent 表示文档处理事件
type DocumentEvent struct {
	DocID    uuid.UUID `json:"doc_id"`
	TenantID uuid.UUID `json:"tenant_id"`
	KBID     uuid.UUID `json:"kb_id"`
	FilePath string    `json:"file_path"`
	MimeType string    `json:"mime_type"`
}

// Worker 处理后台文档处理
type Worker struct {
	docRepo    *models.DocumentRepository
	minio      *minio.Client
	qdrant     *qdrant.Client
	embedding  *embedding.Client
	js         jetstream.JetStream
	consumer   jetstream.Consumer
	parser     *parser.Parser
}

// NewWorker 创建一个新的文档处理 worker
func NewWorker(
	docRepo *models.DocumentRepository,
	minioClient *minio.Client,
	qdrantClient *qdrant.Client,
	embeddingClient *embedding.Client,
	js jetstream.JetStream,
) (*Worker, error) {
	// 创建或获取 documents 流的 consumer
	ctx := context.Background()

	// 使用 EnsureStream 处理竞态条件
	if err := ensureDocumentStream(ctx, js, "documents"); err != nil {
		return nil, fmt.Errorf("确保流存在失败：%w", err)
	}

	// 获取流
	stream, err := js.Stream(ctx, "documents")
	if err != nil {
		return nil, fmt.Errorf("获取流失败：%w", err)
	}

	// 创建 consumer
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       "document-processor",
		FilterSubject: "documents.uploaded",
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxAckPending: 10,
	})
	if err != nil {
		return nil, fmt.Errorf("创建 consumer 失败：%w", err)
	}

	return &Worker{
		docRepo:   docRepo,
		minio:     minioClient,
		qdrant:    qdrantClient,
		embedding: embeddingClient,
		js:        js,
		consumer:  consumer,
		parser:    parser.NewParser(),
	}, nil
}

// ensureDocumentStream 创建 documents 流（如果不存在）
// 处理多个 worker 同时启动时的竞态条件
func ensureDocumentStream(ctx context.Context, js jetstream.JetStream, streamName string) error {
	// 检查流是否存在
	_, err := js.Stream(ctx, streamName)
	if err == nil {
		return nil // 流已存在
	}

	if err != jetstream.ErrStreamNotFound {
		return fmt.Errorf("检查流失败：%w", err)
	}

	// 尝试创建流（如果另一个 worker 同时创建了流，可能会失败）
	_, err = js.CreateStream(ctx, jetstream.StreamConfig{
		Name:      streamName,
		Subjects:  []string{"documents.>"},
		Retention: jetstream.WorkQueuePolicy,
		Storage:   jetstream.FileStorage,
		Replicas:  1,
	})
	if err != nil {
		// 检查流是否被另一个 worker 创建
		if _, getErr := js.Stream(ctx, streamName); getErr == nil {
			return nil // 另一个 worker 已创建
		}
		return fmt.Errorf("创建流失败：%w", err)
	}

	return nil
}

// Start 启动 worker 处理循环
func (w *Worker) Start(ctx context.Context) {
	log.Println("文档处理 worker 已启动")

	maxRetries := 3
	retryCount := make(map[uint64]int)

	for {
		select {
		case <-ctx.Done():
			log.Println("文档处理 worker 已停止")
			return
		default:
			// 获取下一条消息
			batch, err := w.consumer.Fetch(1)
			if err != nil {
				if err == jetstream.ErrNoMessages {
					time.Sleep(1 * time.Second)
					continue
				}
				log.Printf("获取消息失败：%v", err)
				time.Sleep(500 * time.Millisecond)
				continue
			}

			// 遍历批次
			for msg := range batch.Messages() {
				// 获取消息序列号用于跟踪
				meta, err := msg.Metadata()
				if err != nil {
					log.Printf("获取消息元数据失败：%v", err)
					continue
				}
				msgSeq := meta.Sequence.Stream

				// 处理消息
				if err := w.processMessage(ctx, msg); err != nil {
					log.Printf("处理消息失败：%v", err)
					// 跟踪重试次数
					retryCount[msgSeq]++
					if retryCount[msgSeq] >= maxRetries {
						log.Printf("消息超过最大重试次数，确认：%d", msgSeq)
						_ = msg.Ack()
						delete(retryCount, msgSeq)
					} else {
						// Nak 以延迟重新交付
						_ = msg.NakWithDelay(5 * time.Second)
					}
				}
			}
		}
	}
}

// processMessage 处理单个文档事件
func (w *Worker) processMessage(ctx context.Context, msg jetstream.Msg) error {
	var event DocumentEvent
	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		return fmt.Errorf("反序列化事件失败：%w", err)
	}

	log.Printf("处理文档：%s", event.DocID)

	// 更新状态为处理中
	if err := w.docRepo.UpdateStatus(ctx, event.DocID, models.DocumentStatusProcessing, nil); err != nil {
		return fmt.Errorf("更新状态失败：%w", err)
	}

	// 从 MinIO 下载文件
	fileData, err := w.minio.GetFile(ctx, getObjectName(event.FilePath))
	if err != nil {
		errorMsg := fmt.Sprintf("下载文件失败：%v", err)
		_ = w.docRepo.UpdateStatus(ctx, event.DocID, models.DocumentStatusError, &errorMsg)
		return fmt.Errorf("下载文件失败：%w", err)
	}

	// 解析文档
	parseResult, err := w.parser.Parse(fileData, event.MimeType)
	if err != nil {
		errorMsg := fmt.Sprintf("解析文档失败：%v", err)
		_ = w.docRepo.UpdateStatus(ctx, event.DocID, models.DocumentStatusError, &errorMsg)
		return fmt.Errorf("解析文档失败：%w", err)
	}

	log.Printf("文档已解析：%s, 内容长度：%d", event.DocID, len(parseResult.Content))

	// 将内容分块
	log.Printf("开始分块：%s", event.DocID)
	chunks := w.parser.Chunk(parseResult.Content, parser.DefaultChunkOptions())
	log.Printf("文档已分块：%s, 块数：%d", event.DocID, len(chunks))

	// 为所有块生成嵌入向量
	texts := make([]string, len(chunks))
	for i, chunk := range chunks {
		texts[i] = chunk.Content
	}

	log.Printf("开始生成嵌入向量：%s, 文本数：%d", event.DocID, len(texts))
	embeddings, err := w.embedding.GenerateEmbeddings(ctx, texts)
	if err != nil {
		errorMsg := fmt.Sprintf("生成嵌入向量失败：%v", err)
		_ = w.docRepo.UpdateStatus(ctx, event.DocID, models.DocumentStatusError, &errorMsg)
		return fmt.Errorf("生成嵌入向量失败：%w", err)
	}

	log.Printf("嵌入向量已生成：%s, 数量：%d", event.DocID, len(embeddings))

	// 准备 Qdrant 块
	qdrantChunks := make([]qdrant.Chunk, len(chunks))
	for i, chunk := range chunks {
		// 将 map[string]string 转换为 map[string]any
		var metadata map[string]any
		if parseResult.Meta != nil {
			metadata = make(map[string]any, len(parseResult.Meta))
			for k, v := range parseResult.Meta {
				metadata[k] = v
			}
		}

		qdrantChunks[i] = qdrant.Chunk{
			ID:         uuid.New(),
			DocumentID: event.DocID,
			TenantID:   event.TenantID,
			KBID:       event.KBID,
			Content:    chunk.Content,
			ChunkIndex: chunk.Index,
			Embedding:  embeddings[i],
			Metadata:   metadata,
		}
	}

	// 存储块到 Qdrant
	if err := w.qdrant.UpsertChunks(ctx, qdrantChunks); err != nil {
		errorMsg := fmt.Sprintf("存储嵌入向量失败：%v", err)
		_ = w.docRepo.UpdateStatus(ctx, event.DocID, models.DocumentStatusError, &errorMsg)
		return fmt.Errorf("存储嵌入向量失败：%w", err)
	}

	log.Printf("嵌入向量已存储到 Qdrant: %s", event.DocID)

	// 标记为已索引
	if err := w.docRepo.UpdateStatus(ctx, event.DocID, models.DocumentStatusIndexed, nil); err != nil {
		return fmt.Errorf("更新状态失败：%w", err)
	}

	log.Printf("文档索引成功：%s", event.DocID)

	// 确认消息
	return msg.Ack()
}

// getObjectName 从文件路径提取对象名称
// filePath 格式："documents/tenant_id/kb_id/doc_id_filename" (bucket/objectKey)
// minio.GetFile 已经知道 bucket，所以我们需要提取对象键
func getObjectName(filePath string) string {
	// UploadFile 返回的文件路径格式："documents/{objectKey}"
	// 我们需要移除 bucket 前缀 ("documents/") 来获取对象键
	// 找到第一个 "/" 并返回其后的所有内容
	if idx := strings.Index(filePath, "/"); idx != -1 {
		return filePath[idx+1:]
	}
	return filePath
}
