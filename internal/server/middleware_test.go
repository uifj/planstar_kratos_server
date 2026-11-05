package server

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/go-kratos/kratos-layout/internal/biz"
	"github.com/go-kratos/kratos-layout/internal/conf"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/middleware/ratelimit"
	"github.com/go-kratos/kratos/v3/transport"
)

// --- fakes for the biz ports (server layer tested in isolation) ----------

type fakeLimiter struct {
	allowErr error
	calls    int
}

func (f *fakeLimiter) Allow() (func(error), error) {
	f.calls++
	if f.allowErr != nil {
		return nil, f.allowErr
	}
	return func(error) {}, nil
}

type fakeBreaker struct{ failures int }

func (b *fakeBreaker) Allow() error { return nil }
func (b *fakeBreaker) MarkSuccess() {}
func (b *fakeBreaker) MarkFailed()  { b.failures++ }

type fakeBreakerFactory struct{}

func (fakeBreakerFactory) New() biz.CircuitBreaker { return &fakeBreaker{} }

type fakeKeys struct{ key []byte }

func (f fakeKeys) SigningKey() []byte { return f.key }

func allEnabledConfig() *conf.Middleware {
	return &conf.Middleware{
		Logging:                 true,
		Recovery:                true,
		Validate:                true,
		Metadata:                true,
		Ratelimit:               true,
		Metrics:                 true,
		Tracing:                 true,
		Jwt:                     true,
		CircuitBreaker:          true,
		JwtClient:               true,
		PropagatedPrefix:        []string{"x-md-"},
		PublicOperations:        []string{"/todo.v1.TodoService/ListTodos"},
		JwtSigningKey:           "test-key",
		TracerName:              "test-tracer",
		MeterName:               "test-meter",
		RateLimit:               8,
		CircuitBreakerThreshold: 3,
	}
}

func newTestMiddlewares(t *testing.T, c *conf.Middleware, limiter biz.RateLimiter) *Middlewares {
	t.Helper()
	mw, err := NewMiddlewares(c, slog.Default(), limiter, fakeBreakerFactory{}, fakeKeys{key: []byte("test-key")})
	if err != nil {
		t.Fatalf("NewMiddlewares: %v", err)
	}
	return mw
}

func TestNewMiddlewaresAllEnabled(t *testing.T) {
	mw := newTestMiddlewares(t, allEnabledConfig(), &fakeLimiter{})

	// metadata, tracing, metrics, logging, recovery, ratelimit, jwt, validate
	if got, want := len(mw.Server), 8; got != want {
		t.Errorf("Server chain length = %d, want %d", got, want)
	}
	// Streaming skips rate limiting and metrics.
	if got, want := len(mw.Stream), 6; got != want {
		t.Errorf("Stream chain length = %d, want %d", got, want)
	}
	// metadata, tracing, metrics, logging, circuitbreaker, jwt.Client
	if got, want := len(mw.Client), 6; got != want {
		t.Errorf("Client chain length = %d, want %d", got, want)
	}
	if got, want := len(mw.Routes), 1; got != want {
		t.Fatalf("Routes length = %d, want %d", got, want)
	}
	if got, want := mw.Routes[0].Pattern, "/todo.v1.TodoService/*"; got != want {
		t.Errorf("route pattern = %q, want %q", got, want)
	}
}

func TestNewMiddlewaresAllDisabled(t *testing.T) {
	mw := newTestMiddlewares(t, &conf.Middleware{}, &fakeLimiter{})
	if len(mw.Server) != 0 || len(mw.Stream) != 0 || len(mw.Client) != 0 || len(mw.Routes) != 0 {
		t.Fatalf("expected empty chains, got server=%d stream=%d client=%d routes=%d",
			len(mw.Server), len(mw.Stream), len(mw.Client), len(mw.Routes))
	}
}

func TestNewMiddlewaresNilConfigIsSafe(t *testing.T) {
	if _, err := NewMiddlewares(nil, nil, &fakeLimiter{}, fakeBreakerFactory{}, fakeKeys{}); err != nil {
		t.Fatalf("NewMiddlewares(nil config): %v", err)
	}
}

func TestClientMiddlewareComposesChain(t *testing.T) {
	mw := newTestMiddlewares(t, allEnabledConfig(), &fakeLimiter{})
	called := false
	handler := func(_ context.Context, req any) (any, error) {
		called = true
		return "reply", nil
	}
	// circuitbreaker.Client (and the rest of the outbound chain) require a
	// client transport context; installing them elsewhere is unsupported.
	reply, err := mw.ClientMiddleware()(handler)(clientCtx("/todo.v1.TodoService/CreateTodo"), "request")
	if err != nil {
		t.Fatalf("ClientMiddleware: %v", err)
	}
	if !called || reply != "reply" {
		t.Fatalf("handler not reached correctly: called=%v reply=%v", called, reply)
	}
}

func TestAuditCallsNextOnce(t *testing.T) {
	calls := 0
	next := func(_ context.Context, req any) (any, error) {
		calls++
		return "ok", nil
	}
	reply, err := audit(slog.Default())(next)(context.Background(), nil)
	if err != nil || reply != "ok" || calls != 1 {
		t.Fatalf("audit: reply=%v err=%v calls=%d", reply, err, calls)
	}
}

func TestLimiterAdapter(t *testing.T) {
	inner := &fakeLimiter{}
	done, err := limiterAdapter{inner}.Allow()
	if err != nil {
		t.Fatalf("Allow: %v", err)
	}
	done(ratelimit.DoneInfo{Err: nil})
	if inner.calls != 1 {
		t.Fatalf("inner limiter called %d times, want 1", inner.calls)
	}

	rejecting := limiterAdapter{&fakeLimiter{allowErr: biz.ErrRateLimitExceeded}}
	if _, err := rejecting.Allow(); !errors.Is(err, biz.ErrRateLimitExceeded) {
		t.Fatalf("rejecting limiter = %v, want ErrRateLimitExceeded", err)
	}
}

func TestBreakerAdapter(t *testing.T) {
	inner := &fakeBreaker{}
	a := breakerAdapter{inner}
	if err := a.Allow(); err != nil {
		t.Fatalf("Allow: %v", err)
	}
	a.MarkFailed()
	a.MarkSuccess()
	if inner.failures != 1 {
		t.Fatalf("MarkFailed delegated %d times, want 1", inner.failures)
	}
}

// --- transport fakes for the selector + JWT integration test ------------

type testHeader struct{ values map[string][]string }

func newTestHeader() *testHeader { return &testHeader{values: make(map[string][]string)} }
func (h *testHeader) Get(k string) string {
	if v := h.values[k]; len(v) > 0 {
		return v[0]
	}
	return ""
}
func (h *testHeader) Set(k, v string) { h.values[k] = []string{v} }
func (h *testHeader) Add(k, v string) { h.values[k] = append(h.values[k], v) }
func (h *testHeader) Keys() []string {
	keys := make([]string, 0, len(h.values))
	for k := range h.values {
		keys = append(keys, k)
	}
	return keys
}
func (h *testHeader) Values(k string) []string { return h.values[k] }

type testTransporter struct {
	operation string
	header    *testHeader
}

func (t *testTransporter) Kind() transport.Kind            { return transport.KindHTTP }
func (t *testTransporter) Endpoint() string                { return "http://127.0.0.1:8000" }
func (t *testTransporter) Operation() string               { return t.operation }
func (t *testTransporter) RequestHeader() transport.Header { return t.header }
func (t *testTransporter) ReplyHeader() transport.Header   { return t.header }

func serverCtx(operation string) context.Context {
	return transport.NewServerContext(context.Background(), &testTransporter{
		operation: operation,
		header:    newTestHeader(),
	})
}

func clientCtx(operation string) context.Context {
	return transport.NewClientContext(context.Background(), &testTransporter{
		operation: operation,
		header:    newTestHeader(),
	})
}

func TestAuthMiddlewarePublicAllowlist(t *testing.T) {
	c := allEnabledConfig()
	m := authMiddleware(c, fakeKeys{key: []byte("test-key")})
	handler := func(_ context.Context, req any) (any, error) { return "ok", nil }

	// Public operation bypasses JWT entirely.
	reply, err := m(handler)(serverCtx("/todo.v1.TodoService/ListTodos"), nil)
	if err != nil || reply != "ok" {
		t.Fatalf("public operation: reply=%v err=%v", reply, err)
	}

	// Protected operation without a token is rejected by jwt.Server.
	if _, err := m(handler)(serverCtx("/todo.v1.TodoService/CreateTodo"), nil); err == nil {
		t.Fatal("protected operation without token should fail")
	}
}

func TestValidateRequests(t *testing.T) {
	if err := validateRequests(struct{}{}); err != nil {
		t.Fatalf("non-proto request: %v", err)
	}
}

var _ middleware.Middleware = audit(slog.Default())
