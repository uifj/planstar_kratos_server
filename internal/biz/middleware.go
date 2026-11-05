package biz

import "errors"

// This file declares the ports for cross-cutting transport policies.
//
// AGENTS.md establishes the inversion seam as "biz declares the interface,
// data implements it". The same rule applies here: the domain layer owns
// these interfaces and knows nothing about Kratos, protobuf, HTTP, or gRPC.
// internal/server only composes the middleware and internal/data supplies
// the concrete policies, wired together by Wire in cmd/server.
//
// Keeping framework types out of biz means a policy can be replaced (a
// different limiter, breaker or key source) without touching any layer
// that declares or consumes the interface.

var (
	// ErrRateLimitExceeded is returned by RateLimiter.Allow when the request
	// must be rejected. The transport layer maps it to a stable public error.
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	// ErrCircuitOpen is returned by CircuitBreaker.Allow while the breaker
	// refuses outbound calls.
	ErrCircuitOpen = errors.New("circuit breaker open")
)

// RateLimiter is the admission-control port.
//
// Allow returns a completion callback that the caller must invoke exactly
// once with the handler's error. A non-nil error rejects the request; in
// that case the returned callback is ignored.
type RateLimiter interface {
	Allow() (func(error), error)
}

// CircuitBreaker protects outbound calls to a dependency.
type CircuitBreaker interface {
	Allow() error
	MarkSuccess()
	MarkFailed()
}

// CircuitBreakerFactory creates independent breakers, one per canonical RPC
// operation. Sharing mutable state across unrelated operations would change
// the isolation semantics.
type CircuitBreakerFactory interface {
	New() CircuitBreaker
}

// AuthKeyProvider owns the key used to verify inbound tokens. Keeping the
// credential behind a port means the transport layer never reads config or
// secrets directly, and the key source can be rotated or moved to a vault
// without changing the middleware chain.
type AuthKeyProvider interface {
	SigningKey() []byte
}
