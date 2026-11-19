package ops_test

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"planstar/internal/ops"
)

type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{t: time.Unix(1_700_000_000, 0)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

func newReadiness(checks []ops.Check, clock *fakeClock, ttl time.Duration) *ops.Readiness {
	opts := ops.Options{
		Service:      "planstar.api",
		Version:      "v-test",
		TTL:          ttl,
		TotalTimeout: time.Second,
		Now:          clock.Now,
	}
	return ops.NewReadiness(checks, opts)
}

func TestReadinessAllOK(t *testing.T) {
	clock := newFakeClock()
	r := newReadiness([]ops.Check{
		{Name: "mysql", Critical: true, Probe: func(context.Context) error { return nil }},
		{Name: "redis", Critical: true, Probe: func(context.Context) error { return nil }},
	}, clock, 2*time.Second)
	r.MarkStarted()

	resp := r.Check(context.Background())
	if !resp.Ready || resp.Status != ops.StatusOK {
		t.Fatalf("ready=%v status=%q, want ready/ok", resp.Ready, resp.Status)
	}
	if resp.Cached {
		t.Fatal("首次检查不应命中缓存")
	}
	if len(resp.Checks) != 2 {
		t.Fatalf("checks len = %d, want 2", len(resp.Checks))
	}
	if resp.Service != "planstar.api" || resp.Version != "v-test" {
		t.Fatalf("service/version = %q/%q", resp.Service, resp.Version)
	}
	for _, c := range resp.Checks {
		if c.Status != ops.StatusOK {
			t.Fatalf("%s status = %q, want ok", c.Name, c.Status)
		}
	}
}

func TestReadinessCriticalFail(t *testing.T) {
	clock := newFakeClock()
	r := newReadiness([]ops.Check{
		{Name: "mysql", Critical: true, Probe: func(context.Context) error { return errors.New("dial failed") }},
		{Name: "redis", Critical: true, Probe: func(context.Context) error { return nil }},
	}, clock, 2*time.Second)
	r.MarkStarted()

	resp := r.Check(context.Background())
	if resp.Ready || resp.Status != ops.StatusFail {
		t.Fatalf("ready=%v status=%q, want not ready/fail", resp.Ready, resp.Status)
	}
}

func TestReadinessOptionalFailDegraded(t *testing.T) {
	clock := newFakeClock()
	r := newReadiness([]ops.Check{
		{Name: "mysql", Critical: true, Probe: func(context.Context) error { return nil }},
		{Name: "elasticsearch", Critical: false, Probe: func(context.Context) error { return errors.New("es down") }},
	}, clock, 2*time.Second)
	r.MarkStarted()

	resp := r.Check(context.Background())
	if !resp.Ready || resp.Status != ops.StatusDegraded {
		t.Fatalf("ready=%v status=%q, want ready/degraded", resp.Ready, resp.Status)
	}
}

func TestReadinessNilProbeSkipped(t *testing.T) {
	clock := newFakeClock()
	r := newReadiness([]ops.Check{
		{Name: "rabbitmq", Critical: false, Probe: nil},
	}, clock, 2*time.Second)
	r.MarkStarted()

	resp := r.Check(context.Background())
	if !resp.Ready || resp.Status != ops.StatusOK {
		t.Fatalf("ready=%v status=%q, want ready/ok", resp.Ready, resp.Status)
	}
	if got := resp.Checks[0].Status; got != ops.StatusSkipped {
		t.Fatalf("status = %q, want skipped", got)
	}
}

func TestReadinessGate(t *testing.T) {
	clock := newFakeClock()
	var calls atomic.Int64
	r := newReadiness([]ops.Check{
		{Name: "mysql", Critical: true, Probe: func(context.Context) error {
			calls.Add(1)
			return nil
		}},
	}, clock, 2*time.Second)

	before := r.Check(context.Background())
	if before.Ready || before.Status != ops.StatusFail {
		t.Fatalf("启动前 ready=%v status=%q, want not ready/fail", before.Ready, before.Status)
	}
	if calls.Load() != 0 {
		t.Fatalf("启动前不应执行探测，calls=%d", calls.Load())
	}

	r.MarkStarted()
	after := r.Check(context.Background())
	if !after.Ready {
		t.Fatal("启动后应就绪")
	}

	r.MarkStopping()
	stopping := r.Check(context.Background())
	if stopping.Ready || stopping.Status != ops.StatusFail {
		t.Fatalf("停机中 ready=%v status=%q, want not ready/fail", stopping.Ready, stopping.Status)
	}
	if calls.Load() != 1 {
		t.Fatalf("停机闸门不应执行探测，calls=%d", calls.Load())
	}
}

func TestReadinessCacheTTL(t *testing.T) {
	clock := newFakeClock()
	var calls atomic.Int64
	r := newReadiness([]ops.Check{
		{Name: "mysql", Critical: true, Probe: func(context.Context) error {
			calls.Add(1)
			return nil
		}},
	}, clock, 2*time.Second)
	r.MarkStarted()

	first := r.Check(context.Background())
	if first.Cached {
		t.Fatal("首次检查不应命中缓存")
	}
	second := r.Check(context.Background())
	if !second.Cached {
		t.Fatal("TTL 内应命中缓存")
	}
	if calls.Load() != 1 {
		t.Fatalf("TTL 内探测次数 = %d, want 1", calls.Load())
	}

	clock.Advance(3 * time.Second)
	third := r.Check(context.Background())
	if third.Cached {
		t.Fatal("TTL 过期后应重新探测")
	}
	if calls.Load() != 2 {
		t.Fatalf("过期后探测次数 = %d, want 2", calls.Load())
	}
}

func TestReadinessFailureBackoff(t *testing.T) {
	clock := newFakeClock()
	var calls atomic.Int64
	r := newReadiness([]ops.Check{
		{Name: "mysql", Critical: true, Probe: func(context.Context) error {
			calls.Add(1)
			return errors.New("boom")
		}},
	}, clock, 2*time.Second)
	r.MarkStarted()

	r.Check(context.Background())
	// 模拟 100 秒：缓存 2s + 指数退避，探测次数必须远小于 100。
	for i := 0; i < 100; i++ {
		clock.Advance(time.Second)
		r.Check(context.Background())
	}
	if n := calls.Load(); n < 2 || n > 8 {
		t.Fatalf("100 秒内探测次数 = %d, want 2..8（退避生效）", n)
	}
}

func TestReadinessConcurrentSingleflight(t *testing.T) {
	clock := newFakeClock()
	var calls atomic.Int64
	r := newReadiness([]ops.Check{
		{Name: "mysql", Critical: true, Probe: func(context.Context) error {
			calls.Add(1)
			time.Sleep(30 * time.Millisecond)
			return nil
		}},
	}, clock, 2*time.Second)
	r.MarkStarted()

	beforeGoroutines := runtime.NumGoroutine()

	const n = 100
	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make([]ops.ReadyResponse, n)
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			<-start
			results[i] = r.Check(context.Background())
		}(i)
	}
	close(start)
	wg.Wait()

	if got := calls.Load(); got != 1 {
		t.Fatalf("并发探测次数 = %d, want 1（singleflight 去重）", got)
	}
	for i, resp := range results {
		if !resp.Ready {
			t.Fatalf("第 %d 个结果不就绪", i)
		}
	}

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= beforeGoroutines+10 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("疑似 goroutine 泄漏：before=%d after=%d", beforeGoroutines, runtime.NumGoroutine())
}

func TestReadinessSanitizesErrors(t *testing.T) {
	clock := newFakeClock()
	r := newReadiness([]ops.Check{
		{Name: "mysql", Critical: true, Probe: func(context.Context) error {
			return errors.New("dial mysql root:s3cret@tcp(127.0.0.1:3306)/planstar_user failed")
		}},
	}, clock, 2*time.Second)
	r.MarkStarted()

	resp := r.Check(context.Background())
	errText := resp.Checks[0].Error
	if errText == "" {
		t.Fatal("失败检查应返回错误摘要")
	}
	if strings.Contains(errText, "s3cret") {
		t.Fatalf("错误信息未脱敏: %q", errText)
	}
}
