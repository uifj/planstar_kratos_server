package authn

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Revocation 是 jti 撤销表。生产用 Redis（多实例共享），单实例/测试用内存 TTL 表。
type Revocation interface {
	Revoke(ctx context.Context, jti string, ttl time.Duration) error
	// RevokeOnce 原子地标记 jti 已消费：仅首次调用返回 true。
	// refresh 轮换用它替代「先校验后撤销」，消除并发重放窗口（PRODUCTION_AUDIT S3）。
	RevokeOnce(ctx context.Context, jti string, ttl time.Duration) (bool, error)
	IsRevoked(ctx context.Context, jti string) (bool, error)
}

const redisRevokedPrefix = "planstar:authn:revoked:"

type MemoryRevocation struct {
	mu    sync.RWMutex
	items map[string]time.Time
}

func NewMemoryRevocation() *MemoryRevocation {
	return &MemoryRevocation{items: make(map[string]time.Time)}
}

func (m *MemoryRevocation) Revoke(_ context.Context, jti string, ttl time.Duration) error {
	if jti == "" {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for k, until := range m.items {
		if !until.After(now) {
			delete(m.items, k)
		}
	}
	m.items[jti] = now.Add(ttl)
	return nil
}

func (m *MemoryRevocation) RevokeOnce(_ context.Context, jti string, ttl time.Duration) (bool, error) {
	if jti == "" {
		return false, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	if until, ok := m.items[jti]; ok && until.After(now) {
		return false, nil
	}
	m.items[jti] = now.Add(ttl)
	return true, nil
}

func (m *MemoryRevocation) IsRevoked(_ context.Context, jti string) (bool, error) {
	m.mu.RLock()
	until, ok := m.items[jti]
	m.mu.RUnlock()
	if !ok {
		return false, nil
	}
	return until.After(time.Now()), nil
}

// RedisRevocation 用带 TTL 的 key 记录撤销；EXISTS 查询失败时由调用方 fail-closed。
type RedisRevocation struct {
	client *redis.Client
}

func NewRedisRevocation(client *redis.Client) *RedisRevocation {
	return &RedisRevocation{client: client}
}

func (r *RedisRevocation) Revoke(ctx context.Context, jti string, ttl time.Duration) error {
	if r.client == nil || jti == "" {
		return nil
	}
	return r.client.Set(ctx, redisRevokedPrefix+jti, "1", ttl).Err()
}

// RevokeOnce 用 SETNX 原子消费；生产多实例并发刷新时只有第一个请求能成功。
func (r *RedisRevocation) RevokeOnce(ctx context.Context, jti string, ttl time.Duration) (bool, error) {
	if r.client == nil || jti == "" {
		return false, nil
	}
	return r.client.SetNX(ctx, redisRevokedPrefix+jti, "1", ttl).Result()
}

func (r *RedisRevocation) IsRevoked(ctx context.Context, jti string) (bool, error) {
	if r.client == nil {
		return false, nil
	}
	n, err := r.client.Exists(ctx, redisRevokedPrefix+jti).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
