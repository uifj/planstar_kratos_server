package data

import (
	"errors"
	"testing"

	"github.com/go-kratos/kratos-layout/internal/biz"
	"github.com/go-kratos/kratos-layout/internal/conf"
)

func TestConcurrencyLimiter(t *testing.T) {
	l := NewRateLimiter(&conf.Middleware{RateLimit: 1})

	done, err := l.Allow()
	if err != nil {
		t.Fatalf("first Allow: %v", err)
	}
	if _, err := l.Allow(); !errors.Is(err, biz.ErrRateLimitExceeded) {
		t.Fatalf("second Allow = %v, want ErrRateLimitExceeded", err)
	}

	done(nil)
	// A completion callback must be idempotent: an accidental second call
	// must not drive the in-flight counter negative.
	done(errors.New("late"))

	done2, err := l.Allow()
	if err != nil {
		t.Fatalf("Allow after release: %v", err)
	}
	done2(nil)
}

func TestConcurrencyLimiterDefault(t *testing.T) {
	l := NewRateLimiter(nil)
	for i := 0; i < 10; i++ {
		if _, err := l.Allow(); err != nil {
			t.Fatalf("Allow(%d) with default limit: %v", i, err)
		}
	}
}

func TestCircuitBreakerOpensAndResets(t *testing.T) {
	f := NewCircuitBreakerFactory(&conf.Middleware{CircuitBreakerThreshold: 2})
	b := f.New()

	if err := b.Allow(); err != nil {
		t.Fatalf("initial Allow: %v", err)
	}
	b.MarkFailed()
	if err := b.Allow(); err != nil {
		t.Fatalf("Allow after one failure: %v", err)
	}
	b.MarkFailed()
	if err := b.Allow(); !errors.Is(err, biz.ErrCircuitOpen) {
		t.Fatalf("Allow while open = %v, want ErrCircuitOpen", err)
	}

	// A success closes the breaker again.
	b.MarkSuccess()
	if err := b.Allow(); err != nil {
		t.Fatalf("Allow after success: %v", err)
	}
}

func TestCircuitBreakerFactoryIsolatesOperations(t *testing.T) {
	f := NewCircuitBreakerFactory(nil)
	a, b := f.New(), f.New()
	a.MarkFailed()
	a.MarkFailed()
	a.MarkFailed()
	a.MarkFailed()
	a.MarkFailed()
	if err := b.Allow(); err != nil {
		t.Fatalf("independent breaker was affected by another operation: %v", err)
	}
}

func TestAuthKeyProviderReturnsCopy(t *testing.T) {
	p := NewAuthKeyProvider(&conf.Middleware{JwtSigningKey: "secret"})
	first := p.SigningKey()
	first[0] = 'X'
	second := p.SigningKey()
	if string(second) != "secret" {
		t.Fatalf("SigningKey = %q, want %q (must not expose internal slice)", second, "secret")
	}
}
