package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/volcengine/ve-tos-golang-sdk/v2/tos"
	"github.com/volcengine/ve-tos-golang-sdk/v2/tos/enum"

	"planstar/internal/platform/conf"
)

// tosObjectAPI 抽象 TOS 对象操作，测试可注入 fake。
type tosObjectAPI interface {
	PutObjectV2(ctx context.Context, input *tos.PutObjectV2Input) (*tos.PutObjectV2Output, error)
	DeleteObjectV2(ctx context.Context, input *tos.DeleteObjectV2Input) (*tos.DeleteObjectV2Output, error)
	PreSignedURL(input *tos.PreSignedURLInput) (*tos.PreSignedURLOutput, error)
}

type tosStorage struct {
	client     tosObjectAPI
	bucket     string
	cdnHost    string
	presignTTL int64
}

func (s *tosStorage) Put(ctx context.Context, objectKey string, r io.Reader, contentType string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	contentType = resolveContentType(objectKey, contentType)
	if _, err := s.client.PutObjectV2(ctx, &tos.PutObjectV2Input{
		PutObjectBasicInput: tos.PutObjectBasicInput{
			Bucket:      s.bucket,
			Key:         objectKey,
			ContentType: contentType,
		},
		Content: r,
	}); err != nil {
		return "", fmt.Errorf("tos put object: %w", err)
	}
	return s.URL(ctx, objectKey)
}

func (s *tosStorage) Delete(ctx context.Context, objectKey string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := s.client.DeleteObjectV2(ctx, &tos.DeleteObjectV2Input{
		Bucket: s.bucket,
		Key:    objectKey,
	}); err != nil {
		return fmt.Errorf("tos delete object: %w", err)
	}
	return nil
}

func (s *tosStorage) URL(ctx context.Context, objectKey string) (string, error) {
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
	out, err := s.client.PreSignedURL(&tos.PreSignedURLInput{
		HTTPMethod: enum.HttpMethodGet,
		Bucket:     s.bucket,
		Key:        key,
		Expires:    ttl,
	})
	if err != nil {
		return "", fmt.Errorf("tos presign url: %w", err)
	}
	return out.SignedUrl, nil
}

func newTOSStorage(cfg conf.OSS, log *slog.Logger) (ObjectStorage, error) {
	if err := validateTOSConfig(cfg); err != nil {
		return nil, err
	}
	endpoint := strings.TrimSpace(cfg.Endpoint)
	client, err := tos.NewClientV2(endpoint,
		tos.WithRegion(strings.TrimSpace(cfg.Region)),
		tos.WithCredentials(tos.NewStaticCredentials(cfg.AccessKey, cfg.AccessSecret)),
	)
	if err != nil {
		return nil, fmt.Errorf("init tos client: %w", err)
	}
	log.Info("tos object storage enabled", "bucket", cfg.Bucket, "endpoint", endpoint, "region", cfg.Region)
	return &tosStorage{
		client:     client,
		bucket:     strings.TrimSpace(cfg.Bucket),
		cdnHost:    cfg.CDNHost,
		presignTTL: int64(cfg.PresignTTLSeconds),
	}, nil
}

func validateTOSConfig(cfg conf.OSS) error {
	var missing []string
	for _, item := range []struct{ field, value string }{
		{"storage.oss.endpoint", cfg.Endpoint},
		{"storage.oss.region", cfg.Region},
		{"storage.oss.bucket", cfg.Bucket},
		{"storage.oss.access_key", cfg.AccessKey},
		{"storage.oss.access_secret", cfg.AccessSecret},
	} {
		if strings.TrimSpace(item.value) == "" {
			missing = append(missing, item.field)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("tos config missing required fields: %s", strings.Join(missing, ", "))
	}
	return nil
}
