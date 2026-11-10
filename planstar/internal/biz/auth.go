package biz

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"sync"
	"time"

	"planstar/internal/pkg/captcha"
	"planstar/internal/pkg/notify"
	"planstar/internal/pkg/secret"
	"planstar/internal/pkg/session"
)

// 领域错误，service 层据此映射 HTTP 状态。
var (
	ErrBadRequest   = errors.New("bad request")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrRateLimited  = errors.New("rate limited")
	ErrLoginFailed  = errors.New("login failed")
)

const (
	captchaTTL       = 5 * time.Minute
	sendCodeWindow   = 60 * time.Second
	loginFailWindow  = 10 * time.Minute
	loginFailCaptcha = 3
)

// UserAccount 是 biz 的用户主档 DO（不含密码等存储细节的对外投影）。
type UserAccount struct {
	UserID       int64
	UserName     string
	Avatar       string
	BadgeIcon    string
	Bio          string
	Birthday     *time.Time
	Gender       int
	MobileNo     string
	PasswordHash string
	Level        int
	LevelDesc    string
	FollowerNum  int
	FollowingNum int
	WantWatchNum int
	IsDouliUser  bool
	Status       int
}

// ToAuthUser 投影为登录响应结构。
func (u *UserAccount) ToAuthUser() *AuthUser {
	return &AuthUser{
		UserID:       u.UserID,
		UserName:     u.UserName,
		Avatar:       u.Avatar,
		Bio:          u.Bio,
		Gender:       u.Gender,
		Birthday:     BirthTs(u.Birthday),
		MobileNo:     u.MobileNo,
		Level:        u.Level,
		FollowerNum:  u.FollowerNum,
		FollowingNum: u.FollowingNum,
		WantWatchNum: u.WantWatchNum,
		IsDouliUser:  u.IsDouliUser,
		BadgeIcon:    u.BadgeIcon,
	}
}

// AuthRepo 是认证域的仓储端口（data 层实现，落 user_info / login_session / login_code）。
type AuthRepo interface {
	FindUserByMobile(ctx context.Context, mobile string) (*UserAccount, error)
	FindUserByID(ctx context.Context, userID int64) (*UserAccount, error)
	CreateUser(ctx context.Context, mobile string) (*UserAccount, error)
	CreateSession(ctx context.Context, userID int64, loginType int, token string, expireAt time.Time, newUser bool) error
	UserIDByToken(ctx context.Context, token string) (int64, error)
	SaveLoginCode(ctx context.Context, mobile, code string, expireAt time.Time) error
	VerifyLoginCode(ctx context.Context, mobile, code string) bool
}

// AuthCache 是认证域的短期状态端口（data 层用 Redis 实现，失效时回退内存）。
type AuthCache interface {
	SaveSession(ctx context.Context, token string, userID int64, ttl time.Duration) error
	SessionUser(ctx context.Context, token string) (int64, bool, error)
	SaveCaptcha(ctx context.Context, id, code string, ttl time.Duration) error
	ConsumeCaptcha(ctx context.Context, id, code string) (bool, error)
	MarkSend(ctx context.Context, mobile string, window time.Duration) (bool, error)
	FailCount(ctx context.Context, mobile string) (int, error)
	BumpFail(ctx context.Context, mobile string, window time.Duration) error
	ResetFail(ctx context.Context, mobile string) error
}

// AuthUsecase 认证用例：密码 / 短信码登录，会话签发与解析，图形码与风控。
type AuthUsecase struct {
	repo   AuthRepo
	cache  AuthCache
	events EventPublisher
	sms    notify.SMSSender
	log    *slog.Logger
	ttl    time.Duration
}

func NewAuthUsecase(repo AuthRepo, cache AuthCache, events EventPublisher, sms notify.SMSSender, log *slog.Logger, ttl time.Duration) *AuthUsecase {
	if ttl <= 0 {
		ttl = session.SessionTTL
	}
	if events == nil {
		events = NopEventPublisher{}
	}
	if sms == nil {
		sms = notify.NoopSMS{}
	}
	if log == nil {
		log = slog.Default()
	}
	return &AuthUsecase{repo: repo, cache: cache, events: events, sms: sms, log: log, ttl: ttl}
}

// Login 校验凭证并签发会话，返回登录体与 psSession 令牌。
func (uc *AuthUsecase) Login(ctx context.Context, mobile, password, code string) (*LoginBean, string, error) {
	if mobile == "" {
		return nil, "", ErrBadRequest
	}
	user, err := uc.repo.FindUserByMobile(ctx, mobile)
	if err != nil {
		return nil, "", err
	}

	newUser := false
	if code != "" {
		if !uc.repo.VerifyLoginCode(ctx, mobile, code) {
			return nil, "", ErrLoginFailed
		}
		if user == nil {
			user, err = uc.repo.CreateUser(ctx, mobile)
			if err != nil {
				return nil, "", err
			}
			newUser = true
		}
	} else {
		// 用户不存在时仍执行一次等价的口令哈希运算，消除「已注册/未注册」的
		// 响应时序侧信道（避免手机号枚举）。
		hash := dummyPasswordHash()
		if user != nil && user.PasswordHash != "" {
			hash = user.PasswordHash
		}
		valid := secret.VerifyPassword(password, hash)
		if user == nil || user.PasswordHash == "" || !valid {
			return nil, "", ErrLoginFailed
		}
	}
	if user.Status != 1 {
		return nil, "", ErrLoginFailed
	}

	token, err := session.NewToken()
	if err != nil {
		return nil, "", err
	}
	if err := uc.repo.CreateSession(ctx, user.UserID, 11, token, time.Now().Add(uc.ttl), newUser); err != nil {
		return nil, "", err
	}
	if uc.cache != nil {
		if err := uc.cache.SaveSession(ctx, token, user.UserID, uc.ttl); err != nil {
			uc.log.WarnContext(ctx, "cache session failed", "err", err)
		}
	}
	if newUser {
		uc.publish(ctx, NewEvent(EventUserRegistered, user.UserID, map[string]any{
			"mobile":   mobile,
			"userName": user.UserName,
		}))
	}
	return &LoginBean{NeedCaptcha: false, NewUser: newUser, User: user.ToAuthUser()}, token, nil
}

// SendLoginCode 生成并保存短信验证码，再通过 SMSSender 下发（返回明文仅供内部调用方使用）。
func (uc *AuthUsecase) SendLoginCode(ctx context.Context, mobile string) (string, error) {
	if mobile == "" {
		return "", ErrBadRequest
	}
	code, err := loginCode()
	if err != nil {
		// 随机数不可用必须失败关闭，绝不能回退到可预测的固定验证码。
		return "", err
	}
	if err := uc.repo.SaveLoginCode(ctx, mobile, code, time.Now().Add(captchaTTL)); err != nil {
		return "", err
	}
	if err := uc.sms.SendCode(ctx, mobile, code); err != nil {
		uc.log.WarnContext(ctx, "send sms code failed", "mobile", notify.MaskMobile(mobile), "err", err)
		return "", err
	}
	return code, nil
}

// NewCaptcha 生成图形验证码并把 captchaId -> code 写入缓存。
func (uc *AuthUsecase) NewCaptcha(ctx context.Context) (string, []byte, error) {
	id, code, png, err := captcha.New()
	if err != nil {
		return "", nil, err
	}
	if err := uc.cache.SaveCaptcha(ctx, id, code, captchaTTL); err != nil {
		return "", nil, err
	}
	return id, png, nil
}

// VerifyCaptcha 一次性校验图形验证码。
func (uc *AuthUsecase) VerifyCaptcha(ctx context.Context, id, code string) bool {
	if id == "" || code == "" {
		return false
	}
	ok, err := uc.cache.ConsumeCaptcha(ctx, id, code)
	if err != nil {
		uc.log.WarnContext(ctx, "consume captcha failed", "err", err)
		return false
	}
	return ok
}

// AllowSendCode 同一手机号 60s 内只允许发一次（Redis SETNX 保证多实例安全）。
func (uc *AuthUsecase) AllowSendCode(ctx context.Context, mobile string) bool {
	if uc.cache == nil {
		return true
	}
	ok, err := uc.cache.MarkSend(ctx, mobile, sendCodeWindow)
	if err != nil {
		uc.log.WarnContext(ctx, "send code limiter failed", "err", err)
		return false
	}
	return ok
}

// NeedsCaptcha 连续登录失败达到阈值后要求图形码。
func (uc *AuthUsecase) NeedsCaptcha(ctx context.Context, mobile string) bool {
	if uc.cache == nil {
		return false
	}
	n, err := uc.cache.FailCount(ctx, mobile)
	if err != nil {
		uc.log.WarnContext(ctx, "login fail counter failed", "err", err)
		return false
	}
	return n >= loginFailCaptcha
}

// BumpLoginFail 记录一次登录失败。
func (uc *AuthUsecase) BumpLoginFail(ctx context.Context, mobile string) {
	if uc.cache == nil {
		return
	}
	if err := uc.cache.BumpFail(ctx, mobile, loginFailWindow); err != nil {
		uc.log.WarnContext(ctx, "bump login fail failed", "err", err)
	}
}

// ResetLoginFail 登录成功后清空失败计数。
func (uc *AuthUsecase) ResetLoginFail(ctx context.Context, mobile string) {
	if uc.cache == nil {
		return
	}
	if err := uc.cache.ResetFail(ctx, mobile); err != nil {
		uc.log.WarnContext(ctx, "reset login fail failed", "err", err)
	}
}

// ResolveSession 由 psSession 解析登录用户（优先 Redis 缓存，回源 MySQL）。
func (uc *AuthUsecase) ResolveSession(ctx context.Context, token string) (int64, error) {
	if token == "" {
		return 0, ErrUnauthorized
	}
	if uc.cache != nil {
		if userID, ok, err := uc.cache.SessionUser(ctx, token); err == nil && ok {
			return userID, nil
		}
	}
	userID, err := uc.repo.UserIDByToken(ctx, token)
	if err != nil {
		return 0, err
	}
	if userID == 0 {
		return 0, ErrUnauthorized
	}
	if uc.cache != nil {
		if err := uc.cache.SaveSession(ctx, token, userID, uc.ttl); err != nil {
			uc.log.WarnContext(ctx, "cache session failed", "err", err)
		}
	}
	return userID, nil
}

func (uc *AuthUsecase) publish(ctx context.Context, ev *DomainEvent) {
	if err := uc.events.Publish(ctx, ev); err != nil {
		uc.log.WarnContext(ctx, "publish event failed", "event", ev.Type, "err", err)
	}
}

// loginCode 用 crypto/rand 逐位生成 6 位数字验证码，避免取模偏置；
// 任何随机源错误都向上返回，禁止降级为固定值。
func loginCode() (string, error) {
	out := make([]byte, 6)
	for i := range out {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", fmt.Errorf("generate login code: %w", err)
		}
		out[i] = byte('0' + n.Int64())
	}
	return string(out), nil
}

// dummyPasswordHash 提供一次与真实校验等价的 PBKDF2 工作量，用于抹平时序差异。
var (
	dummyHashOnce sync.Once
	dummyHash     string
)

func dummyPasswordHash() string {
	dummyHashOnce.Do(func() {
		h, err := secret.HashPassword("non-registered-user-placeholder")
		if err == nil {
			dummyHash = h
		}
	})
	return dummyHash
}
