package data

import (
	"sync"
	"time"

	"github.com/go-kratos/kratos-layout/internal/biz"
	"github.com/go-kratos/kratos-layout/internal/conf"
)

// This file implements the middleware policy ports declared in biz.
// It is the only place that turns configuration into concrete policy
// objects; internal/server receives the interfaces and never sees config
// values or credentials.

const (
	defaultRateLimit               = 64
	defaultCircuitBreakerThreshold = 5
	defaultCircuitBreakerCooldown  = 10 * time.Second
)

// NewRateLimiter builds the application admission-control policy.
func NewRateLimiter(c *conf.Middleware) biz.RateLimiter {
	limit := int(c.GetRateLimit())
	if limit <= 0 {
		limit = defaultRateLimit
	}
	return &concurrencyLimiter{limit: limit}
}

// concurrencyLimiter is a small, dependency-free concurrency limiter.
// It bounds in-flight requests rather than wall-clock request rate, which
// keeps the admission decision deterministic and cheap. A production
// service can swap in a distributed or tenant-aware limiter without
// changing anything outside this file.
type concurrencyLimiter struct {
	mu       sync.Mutex
	limit    int
	inFlight int
}

func (l *concurrencyLimiter) Allow() (func(error), error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.inFlight >= l.limit {
		return nil, biz.ErrRateLimitExceeded
	}
	l.inFlight++
	var once sync.Once
	return func(error) {
		once.Do(func() {
			l.mu.Lock()
			l.inFlight--
			l.mu.Unlock()
		})
	}, nil
}

// NewCircuitBreakerFactory builds the outbound-dependency protection policy.
func NewCircuitBreakerFactory(c *conf.Middleware) biz.CircuitBreakerFactory {
	threshold := c.GetCircuitBreakerThreshold()
	if threshold <= 0 {
		threshold = defaultCircuitBreakerThreshold
	}
	return &sreBreakerFactory{threshold: threshold, cooldown: defaultCircuitBreakerCooldown}
}

type sreBreakerFactory struct {
	threshold int64
	cooldown  time.Duration
}

func (f *sreBreakerFactory) New() biz.CircuitBreaker {
	return &sreBreaker{threshold: f.threshold, cooldown: f.cooldown}
}

// sreBreaker is a consecutive-failure breaker with a fixed cooldown.
// Once the cooldown elapses the next call is a half-open probe.
type sreBreaker struct {
	mu        sync.Mutex
	threshold int64
	cooldown  time.Duration
	failures  int64
	openUntil time.Time
}

func (b *sreBreaker) Allow() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	if now.Before(b.openUntil) {
		return biz.ErrCircuitOpen
	}
	if !b.openUntil.IsZero() {
		// Cooldown elapsed: enter half-open state with a clean slate.
		b.openUntil = time.Time{}
		b.failures = 0
	}
	return nil
}

func (b *sreBreaker) MarkSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures = 0
	b.openUntil = time.Time{}
}

func (b *sreBreaker) MarkFailed() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if time.Now().Before(b.openUntil) {
		return
	}
	b.failures++
	if b.failures >= b.threshold {
		b.openUntil = time.Now().Add(b.cooldown)
	}
}

// NewAuthKeyProvider exposes the JWT verification key as a biz port.
// The value comes from ${JWT_SIGNING_KEY:} in configs/config.yaml, so no
// credential is ever stored in the repository.
func NewAuthKeyProvider(c *conf.Middleware) biz.AuthKeyProvider {
	return &configAuthKeyProvider{key: []byte(c.GetJwtSigningKey())}
}

type configAuthKeyProvider struct {
	key []byte
}

func (p *configAuthKeyProvider) SigningKey() []byte {
	return append([]byte(nil), p.key...)
}
