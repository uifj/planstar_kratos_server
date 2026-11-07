package observability

import (
	"context"
	"database/sql"
	"encoding/hex"
	"log/slog"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"planstar/internal/ops"
)

// Metrics 汇总 HTTP 层的请求量/延迟/在途请求，以及运维探针与数据库连接池指标。
type Metrics struct {
	Registry *prometheus.Registry

	requests        *prometheus.CounterVec
	duration        *prometheus.HistogramVec
	inflight        *prometheus.GaugeVec
	requestSize     *prometheus.HistogramVec
	responseSize    *prometheus.HistogramVec
	activeConns     prometheus.Gauge
	buildInfo       *prometheus.GaugeVec
	opsProbeStatus  *prometheus.GaugeVec
	opsProbeLatency *prometheus.GaugeVec
	opsProbeCache   prometheus.Counter
	opsAuthFailed   *prometheus.CounterVec
	opsAudit        *prometheus.CounterVec
	dbStats         *dbStatsCollector

	// 采集端自保护：限制 /metrics 并发抓取，超限立即 503，避免抓取风暴拖垮运维端口。
	scrapeMaxInflight int
	scrapeTimeout     time.Duration
	scrapeSem         chan struct{}
	scrapeRejected    prometheus.Counter
}

// NewMetrics 构建 Prometheus 指标集（/metrics 暴露）。
func NewMetrics(version, env string) *Metrics {
	reg := prometheus.NewRegistry()
	reg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	reg.MustRegister(prometheus.NewGoCollector())

	latencyBuckets := []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5}
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
			Buckets:   latencyBuckets,
		}, []string{"method", "route"}),
		inflight: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "planstar",
			Subsystem: "http",
			Name:      "requests_in_flight",
			Help:      "当前处理中的 HTTP 请求数",
		}, []string{"method"}),
		requestSize: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "planstar",
			Subsystem: "http",
			Name:      "request_size_bytes",
			Help:      "HTTP 请求体字节数分布",
			Buckets:   prometheus.ExponentialBuckets(256, 4, 8),
		}, []string{"method", "route"}),
		responseSize: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "planstar",
			Subsystem: "http",
			Name:      "response_size_bytes",
			Help:      "HTTP 响应体字节数分布",
			Buckets:   prometheus.ExponentialBuckets(256, 4, 8),
		}, []string{"method", "route"}),
		activeConns: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "planstar",
			Subsystem: "http",
			Name:      "active_connections",
			Help:      "当前打开的 HTTP 连接数",
		}),
		buildInfo: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "planstar",
			Name:      "build_info",
			Help:      "构建信息（始终为 1）",
		}, []string{"version", "go_version", "env"}),
		opsProbeStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "planstar",
			Subsystem: "ops",
			Name:      "probe_status",
			Help:      "依赖探针状态：1=通过/跳过，0=失败",
		}, []string{"probe", "check"}),
		opsProbeLatency: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "planstar",
			Subsystem: "ops",
			Name:      "probe_duration_seconds",
			Help:      "依赖探针最近一次耗时（秒）",
		}, []string{"probe", "check"}),
		opsProbeCache: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "planstar",
			Subsystem: "ops",
			Name:      "probe_cache_hits_total",
			Help:      "就绪探针命中缓存的次数",
		}),
		opsAuthFailed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "planstar",
			Subsystem: "ops",
			Name:      "auth_failed_total",
			Help:      "运维端点鉴权失败次数，按端点分组",
		}, []string{"endpoint"}),
		opsAudit: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "planstar",
			Subsystem: "ops",
			Name:      "audit_total",
			Help:      "运维操作审计计数，按动作/结果分组",
		}, []string{"action", "result"}),
		dbStats: newDBStatsCollector(),
		scrapeRejected: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "planstar",
			Subsystem: "ops",
			Name:      "metrics_scrape_rejected_total",
			Help:      "因超过并发上限被拒绝的 /metrics 抓取次数",
		}),
	}
	m.buildInfo.WithLabelValues(version, runtime.Version(), env).Set(1)
	m.SetScrapeLimits(8, 5*time.Second)

	reg.MustRegister(
		m.requests, m.duration, m.inflight,
		m.requestSize, m.responseSize, m.activeConns, m.buildInfo,
		m.opsProbeStatus, m.opsProbeLatency, m.opsProbeCache,
		m.opsAuthFailed, m.opsAudit, m.scrapeRejected,
		m.dbStats,
	)
	return m
}

// SetScrapeLimits 配置 /metrics 的并发上限与单次抓取超时；应在服务启动前调用。
func (m *Metrics) SetScrapeLimits(maxInflight int, timeout time.Duration) {
	if maxInflight > 0 {
		m.scrapeMaxInflight = maxInflight
		m.scrapeSem = make(chan struct{}, maxInflight)
	}
	if timeout > 0 {
		m.scrapeTimeout = timeout
	}
}

// Handler 返回 Prometheus 指标端点（启用 OpenMetrics 便于导出 exemplar）。
func (m *Metrics) Handler() http.Handler {
	h := promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{EnableOpenMetrics: true})
	if m.scrapeMaxInflight <= 0 || m.scrapeSem == nil {
		return h
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case m.scrapeSem <- struct{}{}:
			defer func() { <-m.scrapeSem }()
		default:
			m.scrapeRejected.Inc()
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"code":"SERVICE_UNAVAILABLE","message":"metrics 抓取并发超限"}`))
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), m.scrapeTimeout)
		defer cancel()
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ObserveProbe 记录一次探针结果；cached=true 时只累加缓存命中计数。
func (m *Metrics) ObserveProbe(probe string, results []ops.CheckResult, cached bool) {
	if cached {
		m.opsProbeCache.Inc()
		return
	}
	for _, res := range results {
		value := 1.0
		if res.Status == ops.StatusFail {
			value = 0
		}
		m.opsProbeStatus.WithLabelValues(probe, res.Name).Set(value)
		m.opsProbeLatency.WithLabelValues(probe, res.Name).Set(res.LatencyMS / 1000)
	}
}

// IncOpsAuthFailed 记录运维端点鉴权失败。
func (m *Metrics) IncOpsAuthFailed(endpoint string) {
	m.opsAuthFailed.WithLabelValues(endpoint).Inc()
}

// IncOpsAudit 记录运维操作审计结果。
func (m *Metrics) IncOpsAudit(action, result string) {
	m.opsAudit.WithLabelValues(action, result).Inc()
}

// ConnState 维护活跃连接数：New 时 +1，Closed/Hijacked 时 -1。
func (m *Metrics) ConnState(_ net.Conn, state http.ConnState) {
	switch state {
	case http.StateNew:
		m.activeConns.Inc()
	case http.StateClosed, http.StateHijacked:
		m.activeConns.Dec()
	}
}

// RegisterDBStats 注册连接池指标提供者；在抓取时调用，避免常驻后台轮询。
func (m *Metrics) RegisterDBStats(provider func() map[string]sql.DBStats) {
	m.dbStats.setProvider(provider)
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
		if r.ContentLength >= 0 {
			m.requestSize.WithLabelValues(r.Method, route).Observe(float64(r.ContentLength))
		}
		next.ServeHTTP(sw, r)

		elapsed := time.Since(start).Seconds()
		m.requests.WithLabelValues(r.Method, route, strconv.Itoa(sw.status)).Inc()
		m.duration.WithLabelValues(r.Method, route).Observe(elapsed)
		m.responseSize.WithLabelValues(r.Method, route).Observe(float64(sw.bytes))

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
	bytes  int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

// dbStatsCollector 在每次抓取时读取连接池快照，避免额外轮询开销。
type dbStatsCollector struct {
	mu       sync.RWMutex
	provider func() map[string]sql.DBStats

	connections  *prometheus.Desc
	waitCount    *prometheus.Desc
	waitDuration *prometheus.Desc
}

func newDBStatsCollector() *dbStatsCollector {
	return &dbStatsCollector{
		connections: prometheus.NewDesc(
			"planstar_db_connections",
			"数据库连接池连接数，按库与状态分组",
			[]string{"domain", "state"}, nil,
		),
		waitCount: prometheus.NewDesc(
			"planstar_db_wait_count_total",
			"数据库连接池累计等待连接次数",
			[]string{"domain"}, nil,
		),
		waitDuration: prometheus.NewDesc(
			"planstar_db_wait_duration_seconds_total",
			"数据库连接池累计等待时长（秒）",
			[]string{"domain"}, nil,
		),
	}
}

func (c *dbStatsCollector) setProvider(provider func() map[string]sql.DBStats) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.provider = provider
}

func (c *dbStatsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.connections
	ch <- c.waitCount
	ch <- c.waitDuration
}

func (c *dbStatsCollector) Collect(ch chan<- prometheus.Metric) {
	c.mu.RLock()
	provider := c.provider
	c.mu.RUnlock()
	if provider == nil {
		return
	}
	for domain, stats := range provider() {
		ch <- prometheus.MustNewConstMetric(c.connections, prometheus.GaugeValue, float64(stats.OpenConnections), domain, "open")
		ch <- prometheus.MustNewConstMetric(c.connections, prometheus.GaugeValue, float64(stats.Idle), domain, "idle")
		ch <- prometheus.MustNewConstMetric(c.connections, prometheus.GaugeValue, float64(stats.InUse), domain, "in_use")
		ch <- prometheus.MustNewConstMetric(c.connections, prometheus.GaugeValue, float64(stats.MaxOpenConnections), domain, "max")
		ch <- prometheus.MustNewConstMetric(c.waitCount, prometheus.CounterValue, float64(stats.WaitCount), domain)
		ch <- prometheus.MustNewConstMetric(c.waitDuration, prometheus.CounterValue, stats.WaitDuration.Seconds(), domain)
	}
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

// InitTracer 初始化 OTLP/gRPC exporter；endpoint 为空时返回 no-op shutdown。
// endpoint 兼容 `http(s)://host:port` 写法，实际 gRPC target 只保留 host:port。
func InitTracer(endpoint string, insecure bool, env string, ratio float64, serviceName, version string, log *slog.Logger) (func(context.Context) error, error) {
	if strings.TrimSpace(endpoint) == "" {
		return func(context.Context) error { return nil }, nil
	}
	target := endpoint
	if i := strings.Index(target, "://"); i >= 0 {
		target = target[i+3:]
	}
	if i := strings.IndexByte(target, '/'); i >= 0 {
		target = target[:i]
	}
	target = strings.TrimSpace(target)

	opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(target)}
	if insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}
	exp, err := otlptracegrpc.New(context.Background(), opts...)
	if err != nil {
		return nil, err
	}
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			attribute.String("service.name", serviceName),
			attribute.String("service.version", version),
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
	if log != nil {
		log.Info("tracing enabled", "endpoint", target, "insecure", insecure, "env", env, "sample_ratio", ratio)
	}
	return tp.Shutdown, nil
}
