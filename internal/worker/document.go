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

// DocumentEvent represents a document processing event
type DocumentEvent struct {
	DocID    uuid.UUID `json:"doc_id"`
	TenantID uuid.UUID `json:"tenant_id"`
	KBID     uuid.UUID `json:"kb_id"`
	FilePath string    `json:"file_path"`
	MimeType string    `json:"mime_type"`
}

// Worker handles background document processing
type Worker struct {
	docRepo    *models.DocumentRepository
	minio      *minio.Client
	qdrant     *qdrant.Client
	embedding  *embedding.Client
	js         jetstream.JetStream
	consumer   jetstream.Consumer
	parser     *parser.Parser
}

// NewWorker creates a new document processing worker
func NewWorker(
	docRepo *models.DocumentRepository,
	minioClient *minio.Client,
	qdrantClient *qdrant.Client,
	embeddingClient *embedding.Client,
	js jetstream.JetStream,
) (*Worker, error) {
	// Create or get consumer for documents stream
	ctx := context.Background()

	// Use EnsureStream which handles race conditions
	if err := ensureDocumentStream(ctx, js, "documents"); err != nil {
		return nil, fmt.Errorf("failed to ensure stream: %w", err)
	}

	// Get stream
	stream, err := js.Stream(ctx, "documents")
	if err != nil {
		return nil, fmt.Errorf("failed to get stream: %w", err)
	}

	// Create consumer
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       "document-processor",
		FilterSubject: "documents.uploaded",
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxAckPending: 10,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
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

// ensureDocumentStream creates the documents stream if it doesn't exist
// handles race condition when multiple workers start simultaneously
func ensureDocumentStream(ctx context.Context, js jetstream.JetStream, streamName string) error {
	// Check if stream exists
	_, err := js.Stream(ctx, streamName)
	if err == nil {
		return nil // Stream exists
	}

	if err != jetstream.ErrStreamNotFound {
		return fmt.Errorf("failed to check stream: %w", err)
	}

	// Try to create stream (may fail if another worker created it simultaneously)
	_, err = js.CreateStream(ctx, jetstream.StreamConfig{
		Name:      streamName,
		Subjects:  []string{"documents.>"},
		Retention: jetstream.WorkQueuePolicy,
		Storage:   jetstream.FileStorage,
		Replicas:  1,
	})
	if err != nil {
		// Check if stream was created by another worker
		if _, getErr := js.Stream(ctx, streamName); getErr == nil {
			return nil // Another worker created it
		}
		return fmt.Errorf("failed to create stream: %w", err)
	}

	return nil
}

// Start starts the worker processing loop
func (w *Worker) Start(ctx context.Context) {
	log.Println("document worker started")

	maxRetries := 3
	retryCount := make(map[uint64]int)

	for {
		select {
		case <-ctx.Done():
			log.Println("document worker stopped")
			return
		default:
			// Fetch next message
			batch, err := w.consumer.Fetch(1)
			if err != nil {
				if err == jetstream.ErrNoMessages {
					time.Sleep(1 * time.Second)
					continue
				}
				log.Printf("failed to fetch message: %v", err)
				time.Sleep(500 * time.Millisecond)
				continue
			}

			// Iterate over batch
			for msg := range batch.Messages() {
				// Get message sequence number for tracking
				meta, err := msg.Metadata()
				if err != nil {
					log.Printf("failed to get message metadata: %v", err)
					continue
				}
				msgSeq := meta.Sequence.Stream

				// Process message
				if err := w.processMessage(ctx, msg); err != nil {
					log.Printf("failed to process message: %v", err)
					// Track retry count
					retryCount[msgSeq]++
					if retryCount[msgSeq] >= maxRetries {
						log.Printf("message exceeded max retries, acknowledging: %d", msgSeq)
						_ = msg.Ack()
						delete(retryCount, msgSeq)
					} else {
						// Nak for redelivery with delay
						_ = msg.NakWithDelay(5 * time.Second)
					}
				}
			}
		}
	}
}

// processMessage processes a single document event
func (w *Worker) processMessage(ctx context.Context, msg jetstream.Msg) error {
	var event DocumentEvent
	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	log.Printf("processing document: %s", event.DocID)

	// Update status to processing
	if err := w.docRepo.UpdateStatus(ctx, event.DocID, models.DocumentStatusProcessing, nil); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	// Download file from MinIO
	fileData, err := w.minio.GetFile(ctx, getObjectName(event.FilePath))
	if err != nil {
		errorMsg := fmt.Sprintf("failed to download file: %v", err)
		_ = w.docRepo.UpdateStatus(ctx, event.DocID, models.DocumentStatusError, &errorMsg)
		return fmt.Errorf("failed to download file: %w", err)
	}

	// Parse document
	parseResult, err := w.parser.Parse(fileData, event.MimeType)
	if err != nil {
		errorMsg := fmt.Sprintf("failed to parse document: %v", err)
		_ = w.docRepo.UpdateStatus(ctx, event.DocID, models.DocumentStatusError, &errorMsg)
		return fmt.Errorf("failed to parse document: %w", err)
	}

	log.Printf("document parsed: %s, content length: %d", event.DocID, len(parseResult.Content))

	// Chunk the content
	chunks := w.parser.Chunk(parseResult.Content, parser.DefaultChunkOptions())
	log.Printf("document chunked: %s, chunks: %d", event.DocID, len(chunks))

	// Generate embeddings for all chunks
	texts := make([]string, len(chunks))
	for i, chunk := range chunks {
		texts[i] = chunk.Content
	}

	embeddings, err := w.embedding.GenerateEmbeddings(ctx, texts)
	if err != nil {
		errorMsg := fmt.Sprintf("failed to generate embeddings: %v", err)
		_ = w.docRepo.UpdateStatus(ctx, event.DocID, models.DocumentStatusError, &errorMsg)
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	log.Printf("embeddings generated: %s, count: %d", event.DocID, len(embeddings))

	// Prepare chunks for Qdrant
	qdrantChunks := make([]qdrant.Chunk, len(chunks))
	for i, chunk := range chunks {
		// Convert map[string]string to map[string]any
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

	// Store chunks in Qdrant
	if err := w.qdrant.UpsertChunks(ctx, qdrantChunks); err != nil {
		errorMsg := fmt.Sprintf("failed to store embeddings: %v", err)
		_ = w.docRepo.UpdateStatus(ctx, event.DocID, models.DocumentStatusError, &errorMsg)
		return fmt.Errorf("failed to store embeddings: %w", err)
	}

	log.Printf("embeddings stored in qdrant: %s", event.DocID)

	// Mark as indexed
	if err := w.docRepo.UpdateStatus(ctx, event.DocID, models.DocumentStatusIndexed, nil); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	log.Printf("document indexed successfully: %s", event.DocID)

	// Acknowledge message
	return msg.Ack()
}

// getObjectName extracts object name from file path
// filePath format: "documents/tenant_id/kb_id/doc_id_filename" (bucket/objectKey)
// The minio.GetFile already knows the bucket, so we need to extract just the object key
func getObjectName(filePath string) string {
	// File path format from UploadFile: "documents/{objectKey}"
	// We need to remove the bucket prefix ("documents/") to get the object key
	// Find the first "/" and return everything after it
	if idx := strings.Index(filePath, "/"); idx != -1 {
		return filePath[idx+1:]
	}
	return filePath
}
