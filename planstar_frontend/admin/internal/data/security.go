package data

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"planstar/admin/internal/conf"
	"planstar/admin/pkg/auth"
)

const (
	defaultLoginMaxAttempts = 5
	defaultLoginWindow      = 10 * time.Minute
	defaultLoginLock        = 15 * time.Minute

	revokedKeyPrefix = "admin:token:revoked:"
	loginFailPrefix  = "admin:login:fail:"
	loginLockPrefix  = "admin:login:lock:"
)

// revocationStore keeps revoked JWT IDs in Redis until the token would have
// expired anyway, so the keyspace stays bounded.
type revocationStore struct {
	rdb *redis.Client
}

// NewRevocationStore returns the Redis-backed RevocationStore implementation.
func NewRevocationStore(d *Data) auth.RevocationStore {
	return &revocationStore{rdb: d.redis}
}

func (s *revocationStore) Revoke(ctx context.Context, jti string, ttl time.Duration) error {
	if jti == "" || ttl <= 0 {
		return nil
	}
	return s.rdb.Set(ctx, revokedKeyPrefix+jti, "1", ttl).Err()
}

func (s *revocationStore) Revoked(ctx context.Context, jti string) (bool, error) {
	if jti == "" {
		return false, nil
	}
	n, err := s.rdb.Exists(ctx, revokedKeyPrefix+jti).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// attemptLimiter counts failed logins per key and sets a lock key once the
// threshold is crossed. Counting and locking are separate keys so the lock
// survives the shorter window and Reset can clear both.
type attemptLimiter struct {
	rdb    *redis.Client
	max    int
	window time.Duration
	lock   time.Duration
}

// NewAttemptLimiter returns the Redis-backed AttemptLimiter implementation.
func NewAttemptLimiter(d *Data, c *conf.Auth) auth.AttemptLimiter {
	max := int(c.GetLoginMaxAttempts())
	if max <= 0 {
		max = defaultLoginMaxAttempts
	}
	window := c.GetLoginWindow().AsDuration()
	if window <= 0 {
		window = defaultLoginWindow
	}
	lock := c.GetLoginLock().AsDuration()
	if lock <= 0 {
		lock = defaultLoginLock
	}
	return &attemptLimiter{rdb: d.redis, max: max, window: window, lock: lock}
}

func (l *attemptLimiter) Allowed(ctx context.Context, key string) (bool, error) {
	n, err := l.rdb.Exists(ctx, loginLockPrefix+key).Result()
	if err != nil {
		return false, err
	}
	return n == 0, nil
}

func (l *attemptLimiter) Fail(ctx context.Context, key string) error {
	failKey := loginFailPrefix + key
	pipe := l.rdb.TxPipeline()
	count := pipe.Incr(ctx, failKey)
	pipe.Expire(ctx, failKey, l.window)
	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}
	if int(count.Val()) >= l.max {
		return l.rdb.Set(ctx, loginLockPrefix+key, "1", l.lock).Err()
	}
	return nil
}

func (l *attemptLimiter) Reset(ctx context.Context, key string) error {
	return l.rdb.Del(ctx, loginFailPrefix+key, loginLockPrefix+key).Err()
}
