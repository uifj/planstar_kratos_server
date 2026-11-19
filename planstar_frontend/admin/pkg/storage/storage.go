// Package storage provides an object storage port for the admin console.
// Only a Noop implementation exists today; a real OSS/S3 client is wired in
// NewObjectStorage once a provider is selected.
package storage

import (
	"context"
	"io"
	"log/slog"
)

// Config is the storage section of the admin configuration. It is
// intentionally self-contained so this package does not depend on internal/conf.
type Config struct {
	Enabled           bool
	Provider          string
	Endpoint          string
	Region            string
	Bucket            string
	AccessKey         string
	AccessSecret      string
	CDNHost           string
	PresignTTLSeconds int64
}

// ObjectStorage writes and removes objects and resolves their public URLs.
type ObjectStorage interface {
	// Put stores r under objectKey and returns an accessible URL.
	Put(ctx context.Context, objectKey string, r io.Reader, contentType string) (string, error)
	Delete(ctx context.Context, objectKey string) error
	URL(ctx context.Context, objectKey string) (string, error)
}

// Noop discards writes and returns empty URLs. It is used when object storage
// is disabled or when no provider client is wired yet.
type Noop struct{}

func (Noop) Put(context.Context, string, io.Reader, string) (string, error) { return "", nil }

func (Noop) Delete(context.Context, string) error { return nil }

func (Noop) URL(context.Context, string) (string, error) { return "", nil }

// NewObjectStorage builds the object storage client. Until a real provider is
// implemented it always returns Noop; an enabled but unsupported provider is
// logged so the misconfiguration is visible.
func NewObjectStorage(cfg Config, log *slog.Logger) ObjectStorage {
	if log == nil {
		log = slog.Default()
	}
	if cfg.Enabled {
		log.Warn("oss provider not implemented, object storage is disabled", "provider", cfg.Provider)
	}
	return Noop{}
}
