package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	khttp "github.com/go-kratos/kratos/v3/transport/http"

	"planstar/internal/conf"
)

// Health 是依赖健康检查函数（MySQL/Redis/ES 汇总）。
type Health func(context.Context) error

// NewAdminServer 提供运维端点：/metrics、/healthz、/readyz。
// 与业务端口分离，避免指标暴露在公网 API 上。
func NewAdminServer(c conf.Server, metrics http.Handler, health Health, logger *slog.Logger) *khttp.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if err := health(r.Context()); err != nil {
			logger.WarnContext(r.Context(), "readiness check failed", "err", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"unavailable"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})

	srv := khttp.NewServer(
		khttp.Address(c.Admin.Addr),
		khttp.Timeout(10*time.Second),
	)
	srv.HandlePrefix("/", recoverer(logger, mux))
	return srv
}
