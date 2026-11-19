package data

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"planstar/internal/biz"
)

const (
	keySession = "planstar:session:"
	keyCaptcha = "planstar:captcha:"
	keySend    = "planstar:send:"
	keyFail    = "planstar:login:fail:"
)

// AuthCache 用 Redis 保存会话/验证码/风控状态；未配置 Redis 时退化为进程内缓存。
type AuthCache struct {
	d   *Data
	mu  sync.Mutex
	mem map[string]memEntry
}

type memEntry struct {
	value string
	exp   time.Time
}

func NewAuthCache(d *Data) biz.AuthCache {
	return &AuthCache{d: d, mem: map[string]memEntry{}}
}

func (c *AuthCache) redis() *redis.Client { return c.d.Redis() }

func (c *AuthCache) SaveSession(ctx context.Context, token string, userID int64, ttl time.Duration) error {
	return c.set(ctx, keySession+token, strconv.FormatInt(userID, 10), ttl)
}

func (c *AuthCache) SessionUser(ctx context.Context, token string) (int64, bool, error) {
	v, ok, err := c.get(ctx, keySession+token)
	if err != nil || !ok {
		return 0, false, err
	}
	userID, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, false, nil
	}
	return userID, true, nil
}

func (c *AuthCache) SaveCaptcha(ctx context.Context, id, code string, ttl time.Duration) error {
	return c.set(ctx, keyCaptcha+id, code, ttl)
}

func (c *AuthCache) ConsumeCaptcha(ctx context.Context, id, code string) (bool, error) {
	key := keyCaptcha + id
	if rdb := c.redis(); rdb != nil {
		v, err := rdb.GetDel(ctx, key).Result()
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		return v == code, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.mem[key]
	if !ok {
		return false, nil
	}
	delete(c.mem, key)
	if time.Now().After(e.exp) {
		return false, nil
	}
	return e.value == code, nil
}

func (c *AuthCache) MarkSend(ctx context.Context, mobile string, window time.Duration) (bool, error) {
	key := keySend + mobile
	if rdb := c.redis(); rdb != nil {
		return rdb.SetNX(ctx, key, "1", window).Result()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sweep()
	if e, ok := c.mem[key]; ok && time.Now().Before(e.exp) {
		return false, nil
	}
	c.mem[key] = memEntry{value: "1", exp: time.Now().Add(window)}
	return true, nil
}

func (c *AuthCache) FailCount(ctx context.Context, mobile string) (int, error) {
	v, ok, err := c.get(ctx, keyFail+mobile)
	if err != nil || !ok {
		return 0, err
	}
	n, _ := strconv.Atoi(v)
	return n, nil
}

func (c *AuthCache) BumpFail(ctx context.Context, mobile string, window time.Duration) error {
	key := keyFail + mobile
	if rdb := c.redis(); rdb != nil {
		_, err := rdb.TxPipelined(ctx, func(p redis.Pipeliner) error {
			p.Incr(ctx, key)
			p.Expire(ctx, key, window)
			return nil
		})
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sweep()
	e := c.mem[key]
	if !e.exp.IsZero() && time.Now().After(e.exp) {
		e = memEntry{}
	}
	n, _ := strconv.Atoi(e.value)
	e.value = strconv.Itoa(n + 1)
	e.exp = time.Now().Add(window)
	c.mem[key] = e
	return nil
}

func (c *AuthCache) ResetFail(ctx context.Context, mobile string) error {
	key := keyFail + mobile
	if rdb := c.redis(); rdb != nil {
		return rdb.Del(ctx, key).Err()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.mem, key)
	return nil
}

func (c *AuthCache) set(ctx context.Context, key, value string, ttl time.Duration) error {
	if rdb := c.redis(); rdb != nil {
		return rdb.Set(ctx, key, value, ttl).Err()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sweep()
	c.mem[key] = memEntry{value: value, exp: time.Now().Add(ttl)}
	return nil
}

func (c *AuthCache) get(ctx context.Context, key string) (string, bool, error) {
	if rdb := c.redis(); rdb != nil {
		v, err := rdb.Get(ctx, key).Result()
		if errors.Is(err, redis.Nil) {
			return "", false, nil
		}
		if err != nil {
			return "", false, err
		}
		return v, true, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.mem[key]
	if !ok {
		return "", false, nil
	}
	if time.Now().After(e.exp) {
		delete(c.mem, key)
		return "", false, nil
	}
	return e.value, true, nil
}

func (c *AuthCache) sweep() {
	now := time.Now()
	for k, e := range c.mem {
		if now.After(e.exp) {
			delete(c.mem, k)
		}
	}
}
