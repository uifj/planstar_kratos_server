// Package storage 提供对象存储抽象；当前内置 Noop 与阿里云 OSS 实现。
package storage

import (
	"context"
	"io"
	"log/slog"
	"strings"

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

// NewObjectStorage 按配置构造对象存储：enabled=false 返回 Noop；
// provider 为 aliyun/oss 时构造真实客户端，未知 provider 仍降级为 Noop。
func NewObjectStorage(cfg conf.OSS, log *slog.Logger) (ObjectStorage, error) {
	if log == nil {
		log = slog.Default()
	}
	if !cfg.Enabled {
		return Noop{}, nil
	}
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "aliyun", "oss":
		return newAliyunStorage(cfg, log)
	default:
		log.Warn("oss provider not implemented, object storage is disabled", "provider", cfg.Provider)
		return Noop{}, nil
	}
}
