package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"path"
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"

	"planstar/internal/conf"
)

const defaultPresignTTLSeconds int64 = 900

// ossBucketAPI 抽象 Bucket 操作，测试可注入 fake。
type ossBucketAPI interface {
	PutObject(objectKey string, reader io.Reader, options ...oss.Option) error
	DeleteObject(objectKey string, options ...oss.Option) error
	SignURL(objectKey string, method oss.HTTPMethod, expiredInSec int64, options ...oss.Option) (string, error)
}

type aliyunStorage struct {
	bucket     ossBucketAPI
	cdnHost    string
	presignTTL int64
}

func (s *aliyunStorage) Put(ctx context.Context, objectKey string, r io.Reader, contentType string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	contentType = resolveContentType(objectKey, contentType)
	if err := s.bucket.PutObject(objectKey, r, oss.ContentType(contentType)); err != nil {
		return "", fmt.Errorf("aliyun oss put object: %w", err)
	}
	return s.URL(ctx, objectKey)
}

func (s *aliyunStorage) Delete(ctx context.Context, objectKey string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.bucket.DeleteObject(objectKey); err != nil {
		return fmt.Errorf("aliyun oss delete object: %w", err)
	}
	return nil
}

func (s *aliyunStorage) URL(ctx context.Context, objectKey string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	key := strings.TrimPrefix(objectKey, "/")
	if host := strings.TrimSpace(s.cdnHost); host != "" {
		return buildCDNURL(host, key), nil
	}
	ttl := s.presignTTL
	if ttl <= 0 {
		ttl = defaultPresignTTLSeconds
	}
	signed, err := s.bucket.SignURL(key, oss.HTTPGet, ttl)
	if err != nil {
		return "", fmt.Errorf("aliyun oss sign url: %w", err)
	}
	return signed, nil
}

// resolveContentType 优先使用调用方指定值，为空时按扩展名推断。
func resolveContentType(objectKey, contentType string) string {
	if contentType = strings.TrimSpace(contentType); contentType != "" {
		return contentType
	}
	if guess := mime.TypeByExtension(path.Ext(objectKey)); guess != "" {
		return guess
	}
	return "application/octet-stream"
}

// buildCDNURL 统一使用 https 并以单个斜杠拼接 CDN 主机与对象 key。
func buildCDNURL(host, key string) string {
	host = strings.TrimSpace(host)
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimRight(host, "/")
	key = strings.TrimPrefix(key, "/")
	if key == "" {
		return "https://" + host
	}
	return "https://" + host + "/" + key
}

func newAliyunStorage(cfg conf.OSS, log *slog.Logger) (ObjectStorage, error) {
	if err := validateOSSConfig(cfg); err != nil {
		return nil, err
	}
	client, err := oss.New(strings.TrimSpace(cfg.Endpoint), cfg.AccessKey, cfg.AccessSecret)
	if err != nil {
		return nil, fmt.Errorf("init aliyun oss client: %w", err)
	}
	bucket, err := client.Bucket(strings.TrimSpace(cfg.Bucket))
	if err != nil {
		return nil, fmt.Errorf("init aliyun oss bucket: %w", err)
	}
	log.Info("aliyun oss provider enabled", "bucket", cfg.Bucket, "endpoint", cfg.Endpoint)
	return &aliyunStorage{
		bucket:     bucket,
		cdnHost:    cfg.CDNHost,
		presignTTL: int64(cfg.PresignTTLSeconds),
	}, nil
}

func validateOSSConfig(cfg conf.OSS) error {
	var missing []string
	for _, item := range []struct{ field, value string }{
		{"storage.oss.endpoint", cfg.Endpoint},
		{"storage.oss.bucket", cfg.Bucket},
		{"storage.oss.access_key", cfg.AccessKey},
		{"storage.oss.access_secret", cfg.AccessSecret},
	} {
		if strings.TrimSpace(item.value) == "" {
			missing = append(missing, item.field)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("aliyun oss config missing required fields: %s", strings.Join(missing, ", "))
	}
	return nil
}
