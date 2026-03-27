package minio

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
)

// Client wraps MinIO client with application-specific helpers
type Client struct {
	*minio.Client
	log    *zap.Logger
	bucket string
}

// New creates a new MinIO client and ensures bucket exists
func New(endpoint, accessKey, secretKey, bucket string, useSSL bool, log *zap.Logger) (*Client, error) {
	// Create MinIO client
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	log.Info("MinIO client connected", zap.String("endpoint", endpoint))

	c := &Client{
		Client: client,
		log:    log,
		bucket: bucket,
	}

	// Ensure bucket exists
	if err := c.ensureBucket(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ensure bucket: %w", err)
	}

	return c, nil
}

// ensureBucket creates the bucket if it doesn't exist
func (c *Client) ensureBucket(ctx context.Context) error {
	exists, err := c.BucketExists(ctx, c.bucket)
	if err != nil {
		return fmt.Errorf("failed to check bucket: %w", err)
	}

	if exists {
		c.log.Info("MinIO bucket exists", zap.String("bucket", c.bucket))
		return nil
	}

	// Create bucket
	err = c.MakeBucket(ctx, c.bucket, minio.MakeBucketOptions{})
	if err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	c.log.Info("MinIO bucket created", zap.String("bucket", c.bucket))
	return nil
}

// UploadFile uploads a file to MinIO
func (c *Client) UploadFile(ctx context.Context, objectKey string, data []byte, contentType string) (string, error) {
	// Upload with retry logic
	_, err := c.PutObject(
		ctx,
		c.bucket,
		objectKey,
		bytes.NewReader(data),
		int64(len(data)),
		minio.PutObjectOptions{
			ContentType: contentType,
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	c.log.Debug("file uploaded",
		zap.String("bucket", c.bucket),
		zap.String("object", objectKey))

	return fmt.Sprintf("%s/%s", c.bucket, objectKey), nil
}

// GetFile downloads a file from MinIO
func (c *Client) GetFile(ctx context.Context, objectKey string) ([]byte, error) {
	obj, err := c.GetObject(ctx, c.bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer obj.Close()

	// Read all data
	info, err := obj.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat object: %w", err)
	}

	data := make([]byte, info.Size)
	_, err = io.ReadFull(obj, data)
	if err != nil {
		return nil, fmt.Errorf("failed to read object: %w", err)
	}

	return data, nil
}

// DeleteFile deletes a file from MinIO
func (c *Client) DeleteFile(ctx context.Context, objectKey string) error {
	err := c.RemoveObject(ctx, c.bucket, objectKey, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	c.log.Debug("file deleted",
		zap.String("bucket", c.bucket),
		zap.String("object", objectKey))

	return nil
}

// GetPresignedURL generates a presigned URL for temporary access
func (c *Client) GetPresignedURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	url, err := c.PresignedGetObject(ctx, c.bucket, objectKey, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	return url.String(), nil
}
