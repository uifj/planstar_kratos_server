package storage

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"planstar/internal/conf"
)

func TestNewObjectStorageNoopFallback(t *testing.T) {
	cases := []struct {
		name string
		cfg  conf.OSS
	}{
		{name: "disabled", cfg: conf.OSS{}},
		{name: "enabled unsupported provider", cfg: conf.OSS{Enabled: true, Provider: "s3"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
			store, err := NewObjectStorage(tc.cfg, logger)
			if err != nil {
				t.Fatalf("NewObjectStorage() error = %v", err)
			}
			if _, ok := store.(Noop); !ok {
				t.Fatalf("NewObjectStorage() = %T, want Noop", store)
			}

			url, err := store.Put(context.Background(), "a/b.png", strings.NewReader("data"), "image/png")
			if err != nil {
				t.Fatalf("Put() error = %v, want nil", err)
			}
			if url != "" {
				t.Errorf("Put() url = %q, want empty", url)
			}
			if err := store.Delete(context.Background(), "a/b.png"); err != nil {
				t.Fatalf("Delete() error = %v, want nil", err)
			}
			got, err := store.URL(context.Background(), "a/b.png")
			if err != nil {
				t.Fatalf("URL() error = %v, want nil", err)
			}
			if got != "" {
				t.Errorf("URL() = %q, want empty", got)
			}

			if tc.cfg.Enabled && !strings.Contains(buf.String(), "object storage is disabled") {
				t.Errorf("enabled config did not warn, log = %q", buf.String())
			}
		})
	}
}

func TestNewObjectStorageNilLogger(t *testing.T) {
	got, err := NewObjectStorage(conf.OSS{}, nil)
	if err != nil {
		t.Fatalf("NewObjectStorage() error = %v", err)
	}
	if _, ok := got.(Noop); !ok {
		t.Fatalf("NewObjectStorage() = %T, want Noop", got)
	}
}
