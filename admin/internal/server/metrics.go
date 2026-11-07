package server

import (
	"context"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	httpm "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// uuidPattern matches UUIDs anywhere in a request path so they collapse to
// ":id" instead of creating one label value per admin.
var uuidPattern = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)

// Metrics owns the Prometheus registry shared by the HTTP middleware and the
// ops server. Kratos v3 has no first-party metrics middleware, so this is the
// equivalent implementation: requests_total{method,path,code} plus a latency
// histogram over the same normalized labels.
type Metrics struct {
	registry *prometheus.Registry
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
	ready    prometheus.Gauge
}

// NewMetrics builds an isolated registry: only this service's metrics are
// exposed, not the process-wide default collectors.
func NewMetrics() *Metrics {
	m := &Metrics{
		registry: prometheus.NewRegistry(),
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "requests_total",
			Help: "Total HTTP requests handled, by method, normalized path and status code.",
		}, []string{"method", "path", "code"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "duration_seconds",
			Help:    "HTTP request latency in seconds, by method and normalized path.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "path"}),
		ready: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "admin_readyz_ready",
			Help: "Readiness of the admin service as reported by /readyz: 1=ready, 0=not ready.",
		}),
	}
	m.registry.MustRegister(m.requests, m.duration, m.ready)
	m.ready.Set(0)
	return m
}

// SetReady records the latest /readyz outcome.
func (m *Metrics) SetReady(ready bool) {
	if ready {
		m.ready.Set(1)
		return
	}
	m.ready.Set(0)
}

// Handler exposes the registry in the Prometheus text format for /metrics.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// Middleware records one observation per handled request.
func (m *Metrics) Middleware() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (reply any, err error) {
			start := time.Now()
			reply, err = handler(ctx, req)
			method, path := requestLabels(ctx)
			code := "200"
			if se := errors.FromError(err); se != nil {
				code = strconv.Itoa(int(se.Code))
			}
			m.requests.WithLabelValues(method, path, code).Inc()
			m.duration.WithLabelValues(method, path).Observe(time.Since(start).Seconds())
			return reply, err
		}
	}
}

func requestLabels(ctx context.Context) (method, path string) {
	if r, ok := httpm.RequestFromServerContext(ctx); ok {
		return r.Method, normalizePath(r.URL.Path)
	}
	if tr, ok := transport.FromServerContext(ctx); ok {
		return "grpc", tr.Operation()
	}
	return "unknown", "unknown"
}

// normalizePath replaces high-cardinality segments (UUIDs and bare numbers)
// with ":id" so the label set stays bounded.
func normalizePath(path string) string {
	if path == "" {
		return "/"
	}
	path = uuidPattern.ReplaceAllString(path, ":id")
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if segment == "" {
			continue
		}
		if isNumeric(segment) {
			segments[i] = ":id"
		}
	}
	return strings.Join(segments, "/")
}

func isNumeric(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}
