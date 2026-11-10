package server

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	khttp "github.com/go-kratos/kratos/v3/transport/http"

	"planstar/internal/conf"
	"planstar/internal/observability"
	"planstar/internal/pkg/httpx"
)

const (
	// uploadMaxBodyBytes 是上传类路径（/v1/upload 前缀）放宽后的请求体上限。
	uploadMaxBodyBytes int64 = 20 << 20
	// uploadPathPrefix 是上传类路径前缀；当前无 upload 端点，保留常量供后续启用。
	uploadPathPrefix = "/v1/upload"
)

// NewHTTPServer 复用 kratos transport（优雅启停），业务路由通过标准 ServeMux 挂载。
// 链路追踪与 Prometheus 指标由 observability.Middleware 补齐（kratos 中间件不覆盖裸 handler）。
func NewHTTPServer(c conf.Server, handler http.Handler, metrics *observability.Metrics, logger *slog.Logger) *khttp.Server {
	requestTimeout := seconds(c.HTTP.RequestTimeoutSeconds)
	srv := khttp.NewServer(
		khttp.Address(c.HTTP.Addr),
		khttp.Timeout(requestTimeout),
		khttp.Filter(corsFilter(c.HTTP.CORSAllowOrigins)),
	)
	// kratos 未暴露连接级读写/空闲超时选项，直接设置内嵌 http.Server。
	srv.Server.ReadTimeout = seconds(c.HTTP.ReadTimeoutSeconds)
	srv.Server.WriteTimeout = seconds(c.HTTP.WriteTimeoutSeconds)
	srv.Server.IdleTimeout = seconds(c.HTTP.IdleTimeoutSeconds)
	// 显式设置请求头读取超时与最大头长度，避免 Slowloris 与超大头消耗连接。
	srv.Server.ReadHeaderTimeout = 5 * time.Second
	srv.Server.MaxHeaderBytes = 1 << 20
	// 连接级指标：按 New/Closed 事件维护活跃连接数。
	srv.Server.ConnState = metrics.ConnState

	// 由内到外：recoverer → 请求超时 → 请求体限制 → trace 头 → 指标（最外层记录真实状态码）。
	h := recoverer(logger, handler)
	h = timeoutHandler(requestTimeout, logger, h)
	h = bodyLimit(c.HTTP.MaxBodyBytes, logger, h)
	h = traceHeader(h)
	srv.HandlePrefix("/", metrics.Middleware(h, logger))
	return srv
}

// seconds 把配置秒数转为 time.Duration；<=0 时返回 0（表示不启用）。
func seconds(n int) time.Duration {
	if n <= 0 {
		return 0
	}
	return time.Duration(n) * time.Second
}

// recoverer 兜住业务 handler 的 panic，返回统一 JSON 500（kratos 中间件不作用于裸 handler）。
func recoverer(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				if logger != nil {
					// 堆栈只进服务端日志，绝不回显给客户端（阿里手册异常日志规约）。
					logger.ErrorContext(r.Context(), "panic recovered",
						"path", r.URL.Path, "method", r.Method, "panic", rec,
						"stack", string(debug.Stack()))
				}
				httpx.Error(w, r, http.StatusInternalServerError, "INTERNAL", "服务异常")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// timeoutHandler 为单请求套 context 超时；到期返回 504 JSON，并丢弃超时后 handler 的写入。
func timeoutHandler(timeout time.Duration, logger *slog.Logger, next http.Handler) http.Handler {
	if timeout <= 0 {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		tw := &timeoutWriter{w: w, header: make(http.Header), status: http.StatusOK}
		done := make(chan struct{})
		go func() {
			defer close(done)
			next.ServeHTTP(tw, r.WithContext(ctx))
		}()

		select {
		case <-done:
			tw.flush()
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				tw.abandon()
				if logger != nil {
					logger.WarnContext(r.Context(), "request timeout",
						"path", r.URL.Path, "method", r.Method, "timeout", timeout)
				}
				httpx.Error(w, r, http.StatusGatewayTimeout, "TIMEOUT", "请求超时")
				return
			}
			// 客户端主动断开：等 handler 返回后正常收尾。
			<-done
			tw.flush()
		}
	})
}

// timeoutWriter 缓冲 handler 输出，超时后整体丢弃，避免 504 与业务响应混写。
type timeoutWriter struct {
	mu        sync.Mutex
	w         http.ResponseWriter
	header    http.Header
	buf       bytes.Buffer
	status    int
	wrote     bool
	abandoned bool
}

func (tw *timeoutWriter) Header() http.Header { return tw.header }

func (tw *timeoutWriter) WriteHeader(code int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.wrote || tw.abandoned {
		return
	}
	tw.status = code
	tw.wrote = true
}

func (tw *timeoutWriter) Write(b []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.abandoned {
		return 0, http.ErrHandlerTimeout
	}
	if !tw.wrote {
		tw.status = http.StatusOK
		tw.wrote = true
	}
	return tw.buf.Write(b)
}

// abandon 标记超时，后续 handler 写入被忽略。
func (tw *timeoutWriter) abandon() {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.abandoned = true
	tw.buf.Reset()
}

// flush 把缓冲的响应一次性写回客户端。
func (tw *timeoutWriter) flush() {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.abandoned {
		return
	}
	for k, vs := range tw.header {
		for _, v := range vs {
			tw.w.Header().Add(k, v)
		}
	}
	tw.w.WriteHeader(tw.status)
	if tw.buf.Len() > 0 {
		_, _ = tw.w.Write(tw.buf.Bytes())
	}
}

// bodyLimit 限制请求体大小：默认 1 MiB，/v1/upload 前缀放宽到 20 MiB，超限直接 413。
func bodyLimit(maxBytes int64, logger *slog.Logger, next http.Handler) http.Handler {
	if maxBytes <= 0 {
		maxBytes = httpx.DefaultMaxBodyBytes
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit := maxBytes
		if strings.HasPrefix(r.URL.Path, uploadPathPrefix) {
			limit = uploadMaxBodyBytes
		}
		if r.ContentLength > limit {
			if logger != nil {
				logger.WarnContext(r.Context(), "request body too large",
					"path", r.URL.Path, "content_length", r.ContentLength, "limit", limit)
			}
			httpx.Error(w, r, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "请求体过大")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, limit)
		next.ServeHTTP(w, r)
	})
}

// traceHeader 把 span traceId（或随机短串）写入 X-Trace-Id，并注入请求上下文。
func traceHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := httpx.TraceID(r)
		r = r.WithContext(httpx.WithTraceID(r.Context(), id))
		w.Header().Set("X-Trace-Id", id)
		next.ServeHTTP(w, r)
	})
}

// corsFilter 按白名单处理跨域；白名单为空时仅允许同源 Origin。
func corsFilter(allowOrigins []string) khttp.FilterFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, dvid, X-Longitude, X-Latitude")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, HEAD, OPTIONS")
			// 会话令牌只能通过 HttpOnly Cookie 传递，禁止暴露给 JS 可读的响应头。
			w.Header().Set("Access-Control-Expose-Headers", "X-Trace-Id, dvid")
			w.Header().Add("Vary", "Origin")

			if origin := r.Header.Get("Origin"); origin != "" && originAllowed(origin, r, allowOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// originAllowed 判断 Origin 是否在白名单内；白名单为空时按同源（scheme+host）判断。
func originAllowed(origin string, r *http.Request, allowOrigins []string) bool {
	// 仅支持精确白名单，不支持 "*"：带 Cookie 的跨域不能反射任意 Origin。
	if len(allowOrigins) > 0 {
		for _, allowed := range allowOrigins {
			if strings.EqualFold(strings.TrimSpace(allowed), origin) {
				return true
			}
		}
		return false
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	if !strings.EqualFold(u.Host, r.Host) {
		return false
	}
	// 不信任客户端可控的 X-Forwarded-Proto：TLS 在网关终止时，
	// 跨域一律走 cors_allow_origins 精确白名单，避免反射 Allow-Credentials。
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return strings.EqualFold(u.Scheme, scheme)
}
