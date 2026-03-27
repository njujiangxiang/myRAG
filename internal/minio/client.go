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

// Client 封装 MinIO 客户端，提供应用级辅助方法
type Client struct {
	*minio.Client
	log    *zap.Logger
	bucket string
}

// New 创建一个新的 MinIO 客户端并确保存储桶存在
func New(endpoint, accessKey, secretKey, bucket string, useSSL bool, log *zap.Logger) (*Client, error) {
	// 创建 MinIO 客户端
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("创建 MinIO 客户端失败：%w", err)
	}

	log.Info("MinIO 客户端已连接", zap.String("endpoint", endpoint))

	c := &Client{
		Client: client,
		log:    log,
		bucket: bucket,
	}

	// 确保存储桶存在
	if err := c.ensureBucket(context.Background()); err != nil {
		return nil, fmt.Errorf("确保存储桶存在失败：%w", err)
	}

	return c, nil
}

// ensureBucket 创建存储桶（如果不存在）
func (c *Client) ensureBucket(ctx context.Context) error {
	exists, err := c.BucketExists(ctx, c.bucket)
	if err != nil {
		return fmt.Errorf("检查存储桶失败：%w", err)
	}

	if exists {
		c.log.Info("MinIO 存储桶已存在", zap.String("bucket", c.bucket))
		return nil
	}

	// 创建存储桶
	err = c.MakeBucket(ctx, c.bucket, minio.MakeBucketOptions{})
	if err != nil {
		return fmt.Errorf("创建存储桶失败：%w", err)
	}

	c.log.Info("MinIO 存储桶已创建", zap.String("bucket", c.bucket))
	return nil
}

// UploadFile 上传文件到 MinIO
func (c *Client) UploadFile(ctx context.Context, objectKey string, data []byte, contentType string) (string, error) {
	// 上传，带重试逻辑
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
		return "", fmt.Errorf("上传文件失败：%w", err)
	}

	c.log.Debug("文件已上传",
		zap.String("bucket", c.bucket),
		zap.String("object", objectKey))

	return fmt.Sprintf("%s/%s", c.bucket, objectKey), nil
}

// GetFile 从 MinIO 下载文件
func (c *Client) GetFile(ctx context.Context, objectKey string) ([]byte, error) {
	obj, err := c.GetObject(ctx, c.bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取对象失败：%w", err)
	}
	defer obj.Close()

	// 读取所有数据
	info, err := obj.Stat()
	if err != nil {
		return nil, fmt.Errorf("获取对象信息失败：%w", err)
	}

	data := make([]byte, info.Size)
	_, err = io.ReadFull(obj, data)
	if err != nil {
		return nil, fmt.Errorf("读取对象失败：%w", err)
	}

	return data, nil
}

// DeleteFile 从 MinIO 删除文件
func (c *Client) DeleteFile(ctx context.Context, objectKey string) error {
	err := c.RemoveObject(ctx, c.bucket, objectKey, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("删除对象失败：%w", err)
	}

	c.log.Debug("文件已删除",
		zap.String("bucket", c.bucket),
		zap.String("object", objectKey))

	return nil
}

// GetPresignedURL 生成临时访问的预签名 URL
func (c *Client) GetPresignedURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	url, err := c.PresignedGetObject(ctx, c.bucket, objectKey, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("生成预签名 URL 失败：%w", err)
	}

	return url.String(), nil
}
