package server_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"planstar/internal/conf"
	"planstar/internal/observability"
	"planstar/internal/ops"
	"planstar/internal/server"
)

const (
	loadGoroutines   = 100
	loadPerGoroutine = 200
	loadToken        = "ops-load-test-token"
)

type loadStats struct {
	name      string
	latencies []float64
	errors    int64
	qps       float64
	p50       float64
	p95       float64
	p99       float64
	elapsed   time.Duration
}

func TestOpsEndpointsLoad(t *testing.T) {
	metrics := observability.NewMetrics("v-load-test", "test")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	ready := ops.NewReadiness([]ops.Check{
		{Name: "mysql", Critical: true, Probe: noopProbe},
		{Name: "redis", Critical: true, Probe: noopProbe},
		{Name: "elasticsearch", Critical: false, Probe: noopProbe},
		{Name: "rabbitmq", Critical: false},
		{Name: "consul", Critical: false},
	}, ops.Options{
		Service:      "planstar.api",
		Version:      "v-load-test",
		TTL:          2 * time.Second,
		TotalTimeout: time.Second,
	})
	ready.MarkStarted()

	live, readyz := server.NewProbeHandlers(ready, "v-load-test", metrics)
	warmUp(t, live)
	warmUp(t, readyz)

	health := runLoad(t, "healthz", live, loadGoroutines, loadPerGoroutine)
	readiness := runLoad(t, "readyz", readyz, loadGoroutines, loadPerGoroutine)

	assertStats(t, health, 50.0)
	assertStats(t, readiness, 200.0)

	assertMetricsAuth(t, metrics, logger)

	combined := append(append([]float64(nil), health.latencies...), readiness.latencies...)
	totalErrors := health.errors + readiness.errors
	totalElapsed := health.elapsed + readiness.elapsed
	printReport(t, fmt.Sprintf(
		"OPS LOAD TEST qps=%.0f p50=%.1fus p95=%.1fus p99=%.1fus errors=%d error_rate=%.2f%%",
		float64(len(combined))/totalElapsed.Seconds(),
		percentile(combined, 50)*1000, percentile(combined, 95)*1000, percentile(combined, 99)*1000,
		totalErrors, float64(totalErrors)*100/float64(len(combined)),
	))
}

func noopProbe(context.Context) error { return nil }

func warmUp(t *testing.T, handler http.Handler) {
	t.Helper()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://ops.local/warmup", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("warmup 期望 200，实际 %d", rec.Code)
	}
}

func runLoad(t *testing.T, name string, handler http.Handler, goroutines, perGoroutine int) loadStats {
	t.Helper()

	latencies := make([][]float64, goroutines)
	var errors atomic.Int64
	start := time.Now()

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(g int) {
			defer wg.Done()
			lat := make([]float64, perGoroutine)
			for i := 0; i < perGoroutine; i++ {
				req := httptest.NewRequest(http.MethodGet, "http://ops.local/"+name, nil)
				rec := httptest.NewRecorder()
				begin := time.Now()
				handler.ServeHTTP(rec, req)
				lat[i] = float64(time.Since(begin).Microseconds()) / 1000
				if rec.Code != http.StatusOK {
					errors.Add(1)
				}
			}
			latencies[g] = lat
		}(g)
	}
	wg.Wait()

	elapsed := time.Since(start)
	all := make([]float64, 0, goroutines*perGoroutine)
	for _, lat := range latencies {
		all = append(all, lat...)
	}
	stats := loadStats{
		name:      name,
		latencies: all,
		errors:    errors.Load(),
		elapsed:   elapsed,
		qps:       float64(len(all)) / elapsed.Seconds(),
		p50:       percentile(all, 50),
		p95:       percentile(all, 95),
		p99:       percentile(all, 99),
	}
	printReport(t, fmt.Sprintf(
		"OPS LOAD TEST %s qps=%.0f p50=%.1fus p95=%.1fus p99=%.1fus errors=%d error_rate=%.2f%%",
		name, stats.qps, stats.p50*1000, stats.p95*1000, stats.p99*1000,
		stats.errors, float64(stats.errors)*100/float64(len(all)),
	))
	return stats
}

func assertStats(t *testing.T, stats loadStats, p99LimitMS float64) {
	t.Helper()
	if stats.errors != 0 {
		t.Errorf("%s 错误数期望 0，实际 %d（错误率 %.2f%%）", stats.name, stats.errors,
			float64(stats.errors)*100/float64(len(stats.latencies)))
	}
	if stats.p99 >= p99LimitMS {
		t.Errorf("%s P99 期望 < %.0fms，实际 %.2fms", stats.name, p99LimitMS, stats.p99)
	}
}

func assertMetricsAuth(t *testing.T, metrics *observability.Metrics, logger *slog.Logger) {
	t.Helper()
	auth, err := server.NewOpsAuth(conf.Ops{Token: loadToken}, logger, metrics)
	if err != nil {
		t.Fatalf("NewOpsAuth: %v", err)
	}
	handler := auth.Wrap("metrics", metrics.Handler())
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://ops.local/metrics", nil)

	begin := time.Now()
	handler.ServeHTTP(rec, req)
	elapsed := time.Since(begin)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("/metrics 无令牌期望 401，实际 %d", rec.Code)
	}
	if got := rec.Header().Get("WWW-Authenticate"); got != "Bearer" {
		t.Fatalf("/metrics 401 的 WWW-Authenticate 期望 Bearer，实际 %q", got)
	}
	if elapsed > time.Second {
		t.Fatalf("/metrics 无令牌 401 耗时过长：%v", elapsed)
	}
}

func percentile(sortedOrNot []float64, p float64) float64 {
	if len(sortedOrNot) == 0 {
		return 0
	}
	values := append([]float64(nil), sortedOrNot...)
	sort.Float64s(values)
	idx := int(math.Ceil(p/100*float64(len(values)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(values) {
		idx = len(values) - 1
	}
	return values[idx]
}

func printReport(t *testing.T, line string) {
	t.Helper()
	fmt.Println(line)
	t.Log(line)
}
