package server

import (
	"log/slog"
	"net/http"
	"time"

	khttp "github.com/go-kratos/kratos/v3/transport/http"

	"planstar/internal/conf"
	"planstar/internal/observability"
)

// NewHTTPServer 复用 kratos transport（优雅启停），业务路由通过标准 ServeMux 挂载。
// 链路追踪与 Prometheus 指标由 observability.Middleware 补齐（kratos 中间件不覆盖裸 handler）。
func NewHTTPServer(c conf.Server, handler http.Handler, metrics *observability.Metrics, logger *slog.Logger) *khttp.Server {
	srv := khttp.NewServer(
		khttp.Address(c.HTTP.Addr),
		khttp.Timeout(20*time.Second),
		khttp.Filter(corsFilter),
	)
	srv.HandlePrefix("/", metrics.Middleware(recoverer(logger, handler)))
	return srv
}

// recoverer 兜住业务 handler 的 panic（kratos 中间件不作用于 HandlePrefix 注册的裸 handler）。
func recoverer(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.ErrorContext(r.Context(), "panic recovered", "path", r.URL.Path, "panic", rec)
				http.Error(w, "服务异常", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func corsFilter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, dvid, X-Longitude, X-Latitude")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, HEAD, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
