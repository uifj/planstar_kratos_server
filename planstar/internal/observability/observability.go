package observability

import (
	"context"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// Metrics 汇总 HTTP 层的请求量/延迟/在途请求，注册到独立 Registry。
type Metrics struct {
	Registry *prometheus.Registry
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
	inflight *prometheus.GaugeVec
}

// NewMetrics 构建 Prometheus 指标集（/metrics 暴露）。
func NewMetrics() *Metrics {
	reg := prometheus.NewRegistry()
	reg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	reg.MustRegister(prometheus.NewGoCollector())

	m := &Metrics{
		Registry: reg,
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "planstar",
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "HTTP 请求总数，按方法/路由/状态码分组",
		}, []string{"method", "route", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "planstar",
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP 请求耗时分布",
			Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		}, []string{"method", "route"}),
		inflight: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "planstar",
			Subsystem: "http",
			Name:      "requests_in_flight",
			Help:      "当前处理中的 HTTP 请求数",
		}, []string{"method"}),
	}
	reg.MustRegister(m.requests, m.duration, m.inflight)
	return m
}

// Handler 返回 Prometheus 指标端点。
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{})
}

// Middleware 为裸 net/http handler 补链路追踪、指标与访问日志（kratos 中间件不覆盖 ServeMux）。
// logger 为 nil 时只记录指标与 span。
func (m *Metrics) Middleware(next http.Handler, logger *slog.Logger) http.Handler {
	tracer := otel.Tracer("planstar/http")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		route := routeLabel(r)
		ctx, span := tracer.Start(ctx, r.Method+" "+route,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("http.method", r.Method),
				attribute.String("http.target", r.URL.Path),
				attribute.String("http.route", route),
			),
		)
		defer span.End()

		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		rec := newRecorder()
		r = r.WithContext(context.WithValue(ctx, recorderKey{}, rec))

		start := time.Now()
		m.inflight.WithLabelValues(r.Method).Inc()
		defer m.inflight.WithLabelValues(r.Method).Dec()
		next.ServeHTTP(sw, r)

		elapsed := time.Since(start).Seconds()
		m.requests.WithLabelValues(r.Method, route, strconv.Itoa(sw.status)).Inc()
		m.duration.WithLabelValues(r.Method, route).Observe(elapsed)

		// 访问日志：单一入口记录方法、路由、状态码与耗时，字段与指标保持一致。
		if logger != nil {
			logger.InfoContext(ctx, "http request",
				"method", r.Method,
				"route", route,
				"status", sw.status,
				"duration_ms", float64(int(elapsed*1000*1000))/1000,
				"remote_addr", r.RemoteAddr,
			)
		}

		span.SetAttributes(
			attribute.Int("http.status_code", sw.status),
		)
		if rec.err != nil {
			span.RecordError(rec.err)
			span.SetStatus(codes.Error, rec.err.Error())
		}
		if sw.status >= 500 {
			span.SetStatus(codes.Error, http.StatusText(sw.status))
		}
	})
}

func routeLabel(r *http.Request) string {
	if r.Pattern != "" && r.Pattern != "/" {
		return r.Pattern
	}
	// 裸 ServeMux 的 Pattern 在外层包装下不可见，这里按段归一化，避免指标高基数。
	segs := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	for i, seg := range segs {
		if seg == "" {
			continue
		}
		if isID(seg) {
			segs[i] = ":id"
		}
	}
	if len(segs) == 0 || (len(segs) == 1 && segs[0] == "") {
		return "/"
	}
	return "/" + strings.Join(segs, "/")
}

// isID 判定路径段是否为数字 ID 或 16 进制 token。
func isID(seg string) bool {
	if seg == "" {
		return false
	}
	if _, err := strconv.ParseInt(seg, 10, 64); err == nil {
		return true
	}
	if len(seg) >= 16 {
		if _, err := hex.DecodeString(seg); err == nil {
			return true
		}
	}
	return false
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	return w.ResponseWriter.Write(b)
}

type recorderKey struct{}

type recorder struct{ err error }

func newRecorder() *recorder { return &recorder{} }

// RecordError 供 handler 把业务错误附到当前 span（不改变响应）。
func RecordError(ctx context.Context, err error) {
	if err == nil {
		return
	}
	if rec, ok := ctx.Value(recorderKey{}).(*recorder); ok {
		rec.err = err
	}
	if span := trace.SpanFromContext(ctx); span.IsRecording() {
		span.RecordError(err)
	}
}

// NewTraceHandler 给每条日志补 trace_id / span_id，便于日志与链路对齐。
func NewTraceHandler(next slog.Handler) slog.Handler {
	return &traceHandler{Handler: next}
}

type traceHandler struct{ slog.Handler }

func (h *traceHandler) Handle(ctx context.Context, r slog.Record) error {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}
	return h.Handler.Handle(ctx, r)
}

func (h *traceHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &traceHandler{Handler: h.Handler.WithAttrs(attrs)}
}

func (h *traceHandler) WithGroup(name string) slog.Handler {
	return &traceHandler{Handler: h.Handler.WithGroup(name)}
}

// InitTracer 初始化 Jaeger exporter；endpoint 为空时返回 no-op shutdown。
func InitTracer(endpoint, env string, ratio float64, serviceName string, log *slog.Logger) (func(context.Context) error, error) {
	if endpoint == "" {
		return func(context.Context) error { return nil }, nil
	}
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(endpoint)))
	if err != nil {
		return nil, err
	}
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			attribute.String("service.name", serviceName),
			attribute.String("deployment.environment", env),
		),
		resource.WithHost(),
		resource.WithProcess(),
	)
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp, sdktrace.WithBatchTimeout(3*time.Second)),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))
	log.Info("tracing enabled", "endpoint", endpoint, "env", env, "sample_ratio", ratio)
	return tp.Shutdown, nil
}
