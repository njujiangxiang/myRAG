package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
)

// Client wraps NATS JetStream client
type Client struct {
	*nats.Conn
	jetstream.JetStream
	log *zap.Logger
}

// New creates a new NATS JetStream client
func New(url string, log *zap.Logger) (*Client, error) {
	// Connect to NATS
	nc, err := nats.Connect(url,
		nats.ReconnectWait(2*time.Second),
		nats.MaxReconnects(10),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Info("NATS reconnected")
		}),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.Warn("NATS disconnected", zap.Error(err))
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			log.Warn("NATS connection closed")
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream: %w", err)
	}

	log.Info("NATS JetStream connected", zap.String("url", url))

	return &Client{
		Conn:      nc,
		JetStream: js,
		log:       log,
	}, nil
}

// EnsureStream creates the documents stream if it doesn't exist
func (c *Client) EnsureStream(ctx context.Context, streamName string) error {
	// Check if stream exists
	_, err := c.Stream(ctx, streamName)
	if err == nil {
		c.log.Info("NATS stream exists", zap.String("stream", streamName))
		return nil
	}

	if err != jetstream.ErrStreamNotFound {
		return fmt.Errorf("failed to check stream: %w", err)
	}

	// Create stream
	_, err = c.CreateStream(ctx, jetstream.StreamConfig{
		Name:      streamName,
		Subjects:  []string{"documents.>"},
		Retention: jetstream.WorkQueuePolicy,
		Storage:   jetstream.FileStorage,
		Replicas:  1,
	})
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}

	c.log.Info("NATS stream created", zap.String("stream", streamName))
	return nil
}

// PublishDocumentEvent publishes a document processing event
func (c *Client) PublishDocumentEvent(ctx context.Context, eventType string, docID string, data []byte) error {
	subject := fmt.Sprintf("documents.%s", eventType)

	// Use JetStream Publish instead of embedded Conn.Publish
	_, err := c.JetStream.Publish(ctx, subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	c.log.Debug("document event published",
		zap.String("subject", subject),
		zap.String("doc_id", docID))

	return nil
}

// Close gracefully closes the NATS connection
func (c *Client) Close() {
	c.log.Info("closing NATS connection")
	c.Conn.Close()
}
