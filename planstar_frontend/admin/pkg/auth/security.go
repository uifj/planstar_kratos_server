package auth

import (
	"context"
	"time"
)

// RevocationStore records revoked token IDs (JWT `jti`) until they expire.
// The middleware consults it on every authenticated request so a logout takes
// effect immediately instead of waiting for the token to age out.
type RevocationStore interface {
	Revoke(ctx context.Context, jti string, ttl time.Duration) error
	Revoked(ctx context.Context, jti string) (bool, error)
}

// AttemptLimiter counts failed login attempts per identity + client IP and
// locks the key once the configured threshold is reached.
type AttemptLimiter interface {
	Allowed(ctx context.Context, key string) (bool, error)
	Fail(ctx context.Context, key string) error
	Reset(ctx context.Context, key string) error
}
