package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
)

// Client 封装 NATS JetStream 客户端
type Client struct {
	*nats.Conn
	jetstream.JetStream
	log *zap.Logger
}

// New 创建一个新的 NATS JetStream 客户端
func New(url string, log *zap.Logger) (*Client, error) {
	// 连接到 NATS
	nc, err := nats.Connect(url,
		nats.ReconnectWait(2*time.Second),
		nats.MaxReconnects(10),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Info("NATS 重连成功")
		}),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.Warn("NATS 断开连接", zap.Error(err))
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			log.Warn("NATS 连接已关闭")
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("连接 NATS 失败：%w", err)
	}

	// 创建 JetStream 上下文
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("创建 JetStream 失败：%w", err)
	}

	log.Info("NATS JetStream 已连接", zap.String("url", url))

	return &Client{
		Conn:      nc,
		JetStream: js,
		log:       log,
	}, nil
}

// EnsureStream 创建 documents 流（如果不存在）
func (c *Client) EnsureStream(ctx context.Context, streamName string) error {
	// 检查流是否存在
	_, err := c.Stream(ctx, streamName)
	if err == nil {
		c.log.Info("NATS 流已存在", zap.String("stream", streamName))
		return nil
	}

	if err != jetstream.ErrStreamNotFound {
		return fmt.Errorf("检查流失败：%w", err)
	}

	// 创建流
	_, err = c.CreateStream(ctx, jetstream.StreamConfig{
		Name:      streamName,
		Subjects:  []string{"documents.>"},
		Retention: jetstream.WorkQueuePolicy,
		Storage:   jetstream.FileStorage,
		Replicas:  1,
	})
	if err != nil {
		return fmt.Errorf("创建流失败：%w", err)
	}

	c.log.Info("NATS 流已创建", zap.String("stream", streamName))
	return nil
}

// PublishDocumentEvent 发布文档处理事件
func (c *Client) PublishDocumentEvent(ctx context.Context, eventType string, docID string, data []byte) error {
	subject := fmt.Sprintf("documents.%s", eventType)

	// 使用 JetStream Publish 而不是内置的 Conn.Publish
	_, err := c.JetStream.Publish(ctx, subject, data)
	if err != nil {
		return fmt.Errorf("发布事件失败：%w", err)
	}

	c.log.Debug("文档事件已发布",
		zap.String("subject", subject),
		zap.String("doc_id", docID))

	return nil
}

// Close 优雅地关闭 NATS 连接
func (c *Client) Close() {
	c.log.Info("关闭 NATS 连接")
	c.Conn.Close()
}
