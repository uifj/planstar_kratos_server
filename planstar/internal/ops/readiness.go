// Package ops 提供运维探针的检查模型与就绪判定引擎。
// 设计目标：把 Kubernetes 语义（liveness / readiness）与依赖探测解耦，
// 避免单个慢依赖拖垮 /readyz，也避免探针风暴打到数据库。
package ops

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

// 检查状态取值。
const (
	StatusOK       = "ok"
	StatusDegraded = "degraded"
	StatusFail     = "fail"
	StatusSkipped  = "skipped"
)

const (
	defaultCheckTimeout = 2 * time.Second
	defaultCacheTTL     = 2 * time.Second
	defaultTotalTimeout = 3 * time.Second
	maxBackoff          = 30 * time.Second
)

// CheckResult 是单个依赖探测的结果。
type CheckResult struct {
	Name      string  `json:"name"`
	Critical  bool    `json:"critical"`
	Status    string  `json:"status"` // ok|fail|skipped
	LatencyMS float64 `json:"latency_ms"`
	Error     string  `json:"error,omitempty"`
}

// LiveResponse 是 /healthz 的响应：只表示进程存活，不检查依赖。
type LiveResponse struct {
	Service       string    `json:"service"`
	Version       string    `json:"version"`
	Status        string    `json:"status"`
	Timestamp     time.Time `json:"timestamp"`
	UptimeSeconds int64     `json:"uptime_seconds"`
}

// ReadyResponse 是 /readyz 的响应：仅当所有关键依赖通过且未处于停机时才 Ready。
type ReadyResponse struct {
	Service       string        `json:"service"`
	Version       string        `json:"version"`
	Status        string        `json:"status"` // ok|degraded|fail
	Ready         bool          `json:"ready"`
	Timestamp     time.Time     `json:"timestamp"`
	CheckedAt     time.Time     `json:"checked_at"`
	Cached        bool          `json:"cached"`
	UptimeSeconds int64         `json:"uptime_seconds"`
	Checks        []CheckResult `json:"checks"`
}

// Check 描述一个依赖探测项；Probe 为 nil 时该项固定为 skipped，不影响就绪。
type Check struct {
	Name     string
	Critical bool
	Timeout  time.Duration
	Probe    func(context.Context) error
}

// Options 是 Readiness 的可调参数。
type Options struct {
	Service      string
	Version      string
	TTL          time.Duration
	TotalTimeout time.Duration
	Now          func() time.Time
	OnResult     func([]CheckResult)
}

type checkState struct {
	failures int
	retryAt  time.Time
	last     CheckResult
}

// Readiness 并发执行依赖探测，带整体缓存、singleflight 去重与失败退避。
type Readiness struct {
	service      string
	version      string
	checks       []Check
	ttl          time.Duration
	totalTimeout time.Duration
	now          func() time.Time
	onResult     func([]CheckResult)

	started   atomic.Bool
	stopping  atomic.Bool
	startedAt atomic.Int64 // Unix 秒

	mu      sync.Mutex
	cached  *ReadyResponse
	expires time.Time
	states  map[string]*checkState

	sf singleflight.Group
}

// NewReadiness 构造就绪引擎；未配置的字段使用默认值（缓存 2s、总超时 3s、单检查 2s）。
func NewReadiness(checks []Check, opts Options) *Readiness {
	if opts.TTL <= 0 {
		opts.TTL = defaultCacheTTL
	}
	if opts.TotalTimeout <= 0 {
		opts.TotalTimeout = defaultTotalTimeout
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	return &Readiness{
		service:      opts.Service,
		version:      opts.Version,
		checks:       append([]Check(nil), checks...),
		ttl:          opts.TTL,
		totalTimeout: opts.TotalTimeout,
		now:          opts.Now,
		onResult:     opts.OnResult,
		states:       make(map[string]*checkState),
	}
}

// MarkStarted 打开就绪闸门并记录启动时间。
func (r *Readiness) MarkStarted() {
	if r.started.CompareAndSwap(false, true) {
		r.startedAt.Store(r.now().Unix())
	}
}

// MarkStopping 关闭就绪闸门：进程进入优雅停机后即使依赖正常也返回 503。
func (r *Readiness) MarkStopping() {
	r.stopping.Store(true)
}

// Live 返回存活响应，始终为 ok。
func (r *Readiness) Live() LiveResponse {
	now := r.now()
	return LiveResponse{
		Service:       r.service,
		Version:       r.version,
		Status:        StatusOK,
		Timestamp:     now,
		UptimeSeconds: r.uptime(now),
	}
}

// Check 返回当前就绪状态；命中缓存时 Cached=true，且不会触发 OnResult。
func (r *Readiness) Check(ctx context.Context) ReadyResponse {
	if !r.started.Load() || r.stopping.Load() {
		now := r.now()
		return ReadyResponse{
			Service:       r.service,
			Version:       r.version,
			Status:        StatusFail,
			Ready:         false,
			Timestamp:     now,
			CheckedAt:     now,
			UptimeSeconds: r.uptime(now),
		}
	}
	if resp, ok := r.cachedResponse(); ok {
		return resp
	}

	v, _, _ := r.sf.Do("readiness", func() (any, error) {
		// 并发进入时只允许一次真实刷新。
		if resp, ok := r.cachedResponse(); ok {
			return resp, nil
		}
		resp, results := r.evaluate(ctx)
		r.mu.Lock()
		r.cached = &resp
		r.expires = r.now().Add(r.ttl)
		r.mu.Unlock()
		if r.onResult != nil {
			r.onResult(results)
		}
		return resp, nil
	})
	return v.(ReadyResponse)
}

func (r *Readiness) cachedResponse() (ReadyResponse, bool) {
	now := r.now()
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cached == nil || !now.Before(r.expires) {
		return ReadyResponse{}, false
	}
	resp := *r.cached
	resp.Cached = true
	return resp, true
}

// evaluate 并发执行全部检查，受总超时约束；结果按配置顺序返回。
func (r *Readiness) evaluate(ctx context.Context) (ReadyResponse, []CheckResult) {
	start := r.now()
	results := make([]CheckResult, len(r.checks))

	totalCtx, cancel := context.WithTimeout(ctx, r.totalTimeout)
	defer cancel()

	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)
	for i, chk := range r.checks {
		wg.Add(1)
		go func(i int, chk Check) {
			defer wg.Done()
			res := r.runCheck(totalCtx, chk)
			mu.Lock()
			results[i] = res
			mu.Unlock()
		}(i, chk)
	}
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-totalCtx.Done():
	}

	out := make([]CheckResult, len(results))
	mu.Lock()
	for i, chk := range r.checks {
		if results[i].Status == "" {
			out[i] = CheckResult{Name: chk.Name, Critical: chk.Critical, Status: StatusFail, Error: "probe timed out"}
			continue
		}
		out[i] = results[i]
	}
	mu.Unlock()

	status := StatusOK
	for _, res := range out {
		if res.Status != StatusFail {
			continue
		}
		if res.Critical {
			status = StatusFail
			break
		}
		status = StatusDegraded
	}

	now := r.now()
	return ReadyResponse{
		Service:       r.service,
		Version:       r.version,
		Status:        status,
		Ready:         status != StatusFail,
		Timestamp:     start,
		CheckedAt:     now,
		UptimeSeconds: r.uptime(now),
		Checks:        out,
	}, out
}

func (r *Readiness) runCheck(ctx context.Context, chk Check) CheckResult {
	if chk.Probe == nil {
		return CheckResult{Name: chk.Name, Critical: chk.Critical, Status: StatusSkipped}
	}

	now := r.now()
	r.mu.Lock()
	st := r.states[chk.Name]
	if st == nil {
		st = &checkState{}
		r.states[chk.Name] = st
	}
	if st.failures > 0 && now.Before(st.retryAt) {
		res := st.last
		r.mu.Unlock()
		return res
	}
	r.mu.Unlock()

	timeout := chk.Timeout
	if timeout <= 0 {
		timeout = defaultCheckTimeout
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := r.now()
	err := chk.Probe(cctx)
	latency := float64(r.now().Sub(start).Microseconds()) / 1000

	res := CheckResult{Name: chk.Name, Critical: chk.Critical, Status: StatusOK, LatencyMS: latency}
	if err != nil {
		res.Status = StatusFail
		res.Error = sanitizeError(err)
	}

	r.mu.Lock()
	st = r.states[chk.Name]
	if res.Status == StatusFail {
		st.failures++
		st.last = res
		st.retryAt = r.now().Add(backoff(st.failures))
	} else {
		st.failures = 0
		st.retryAt = time.Time{}
		st.last = res
	}
	r.mu.Unlock()
	return res
}

// backoff 按连续失败次数给出退避时长：2s、4s、8s…最多 30s。
func backoff(failures int) time.Duration {
	if failures <= 0 {
		return 0
	}
	d := 2 * time.Second
	for i := 1; i < failures && d < maxBackoff; i++ {
		d *= 2
	}
	if d > maxBackoff {
		d = maxBackoff
	}
	return d
}

func (r *Readiness) uptime(now time.Time) int64 {
	start := r.startedAt.Load()
	if start == 0 {
		return 0
	}
	if d := now.Unix() - start; d > 0 {
		return d
	}
	return 0
}

var (
	urlUserInfoRe = regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9+.-]*://)[^@/\s]+@`)
	dsnUserInfoRe = regexp.MustCompile(`[^\s:@/]+:[^\s@/]*@`)
	passwordRe    = regexp.MustCompile(`(?i)(password|passwd|pwd)\s*=\s*[^\s&;]+`)
)

// sanitizeError 保留简短错误信息，脱敏 DSN/URL 中的用户名口令，避免泄露连接串。
func sanitizeError(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.ReplaceAll(err.Error(), "\n", " ")
	msg = urlUserInfoRe.ReplaceAllString(msg, "${1}***@")
	msg = dsnUserInfoRe.ReplaceAllString(msg, "***@")
	msg = passwordRe.ReplaceAllString(msg, "${1}=***")
	const maxLen = 200
	if len(msg) > maxLen {
		msg = msg[:maxLen] + "..."
	}
	return strings.TrimSpace(msg)
}
