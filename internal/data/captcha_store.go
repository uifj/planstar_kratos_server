package data

import (
	"context"
	"strings"
	"time"

	"github.com/go-kratos/kratos-layout/internal/biz"
	"github.com/go-kratos/kratos-layout/internal/conf"
	"github.com/mojocn/base64Captcha"
	"github.com/redis/go-redis/v9"
)

const defaultCaptchaTTL = 5 * time.Minute

func captchaKey(id string) string { return "mall:captcha:" + id }

// redisCaptchaBackend adapts base64Captcha's synchronous Store interface to
// the request-scoped redis client. An instance is created per request so the
// context flows into every redis call.
type redisCaptchaBackend struct {
	rdb redis.UniversalClient
	ttl time.Duration
	ctx context.Context
}

func (s *redisCaptchaBackend) Set(id, value string) error {
	return s.rdb.Set(s.ctx, captchaKey(id), value, s.ttl).Err()
}

func (s *redisCaptchaBackend) Get(id string, clear bool) string {
	v, err := s.rdb.Get(s.ctx, captchaKey(id)).Result()
	if err != nil {
		return ""
	}
	if clear {
		_ = s.rdb.Del(s.ctx, captchaKey(id)).Err()
	}
	return v
}

func (s *redisCaptchaBackend) Verify(id, answer string, clear bool) bool {
	got := strings.TrimSpace(s.Get(id, clear))
	return got != "" && strings.EqualFold(got, strings.TrimSpace(answer))
}

type captchaStore struct {
	rdb redis.UniversalClient
	ttl time.Duration
}

// NewCaptchaStore keeps captcha answers in Redis instead of the source
// project's in-process memory map, so the service is horizontally scalable.
func NewCaptchaStore(d *Data, m *conf.Mall) biz.CaptchaStore {
	ttl := defaultCaptchaTTL
	if m != nil {
		if b := m.GetBusiness(); b != nil {
			if v := b.GetCaptchaTtl(); v != nil && v.AsDuration() > 0 {
				ttl = v.AsDuration()
			}
		}
	}
	return &captchaStore{rdb: d.rdb, ttl: ttl}
}

func (c *captchaStore) Create(ctx context.Context) (string, string, error) {
	driver := base64Captcha.NewDriverDigit(80, 240, 5, 0.7, 80)
	backend := &redisCaptchaBackend{rdb: c.rdb, ttl: c.ttl, ctx: ctx}
	// Generate() calls backend.Set(id, answer) internally; b64s is already a
	// complete "data:image/png;base64,..." data URL.
	id, image, _, err := base64Captcha.NewCaptcha(driver, backend).Generate()
	if err != nil {
		return "", "", err
	}
	return id, image, nil
}

func (c *captchaStore) Verify(ctx context.Context, id, value string) (bool, error) {
	backend := &redisCaptchaBackend{rdb: c.rdb, ttl: c.ttl, ctx: ctx}
	// clear=true makes a captcha single-use, even when the answer is wrong.
	return backend.Verify(id, value, true), nil
}
