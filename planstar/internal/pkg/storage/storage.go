// Package storage 提供对象存储抽象；当前仅内置 Noop 实现，
// 真实服务商（OSS / S3 等）客户端可在 NewObjectStorage 中按 provider 接入。
package storage

import (
	"context"
	"io"
	"log/slog"

	"planstar/internal/conf"
)

// ObjectStorage 是对象存储端口。
type ObjectStorage interface {
	// Put 写入对象并返回可访问地址。
	Put(ctx context.Context, objectKey string, r io.Reader, contentType string) (string, error)
	Delete(ctx context.Context, objectKey string) error
	URL(ctx context.Context, objectKey string) (string, error)
}

// Noop 丢弃写入并返回空地址，用于对象存储未启用时。
type Noop struct{}

func (Noop) Put(context.Context, string, io.Reader, string) (string, error) { return "", nil }
func (Noop) Delete(context.Context, string) error                           { return nil }
func (Noop) URL(context.Context, string) (string, error)                    { return "", nil }

// NewObjectStorage 按配置构造对象存储：enabled=false 或 provider 尚未接入时返回 Noop。
func NewObjectStorage(cfg conf.OSS, log *slog.Logger) ObjectStorage {
	if log == nil {
		log = slog.Default()
	}
	if cfg.Enabled {
		log.Warn("oss provider not implemented, object storage is disabled", "provider", cfg.Provider)
	}
	return Noop{}
}
