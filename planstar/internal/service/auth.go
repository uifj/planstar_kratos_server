package service

import (
	"errors"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"planstar/internal/biz"
	"planstar/internal/observability"
	"planstar/internal/pkg/httpx"
	"planstar/internal/pkg/session"
)

var mobileRe = regexp.MustCompile(`^1\d{10}$`)

type loginRequest struct {
	UA      string `json:"ua"`
	CT      int    `json:"ct"`
	PW      string `json:"pw"`
	Code    string `json:"code"`
	Captcha string `json:"captcha"`
}

type sendCodeRequest struct {
	UA      string `json:"ua"`
	Captcha string `json:"captcha"`
}

// captchaImage 生成 image/png 图形码，明文码存 Redis，captchaId 写入 psCaptcha Cookie。
func (s *Service) captchaImage(w http.ResponseWriter, r *http.Request) {
	id, png, err := s.auth.NewCaptcha(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: "psCaptcha", Value: id, Path: "/", HttpOnly: true, MaxAge: 300, SameSite: http.SameSiteLaxMode,
	})
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(png)
}

// sendLoginCode 契约：发送结果恒 200 + {success, needCaptcha, errorMsg}；触发限流时返回 429。
func (s *Service) sendLoginCode(w http.ResponseWriter, r *http.Request) {
	var body sendCodeRequest
	if err := httpx.ReadJSON(r, &body); err != nil {
		if errors.Is(err, httpx.ErrBodyTooLarge) {
			s.fail(w, r, err)
			return
		}
		httpx.JSON(w, http.StatusOK, &biz.SendCodeResp{Success: false, ErrorMsg: "请求格式错误"})
		return
	}
	mobile := strings.TrimSpace(body.UA)
	if !mobileRe.MatchString(mobile) {
		httpx.JSON(w, http.StatusOK, &biz.SendCodeResp{Success: false, ErrorMsg: "手机号格式不正确"})
		return
	}
	if body.Captcha != "" && !s.verifyCaptcha(r, body.Captcha) {
		httpx.JSON(w, http.StatusOK, &biz.SendCodeResp{Success: false, NeedCaptcha: true, ErrorMsg: "图形验证码错误"})
		return
	}

	cooldown, ipLimit, _, _ := s.authSettings()
	ok, reason, retryAfter := s.limit.allowSend(mobile, clientIP(r), cooldown, ipLimit)
	if !ok {
		s.log.WarnContext(r.Context(), "send code rate limited",
			"mobile", maskMobile(mobile), "ip", clientIP(r), "dimension", reason, "retry_after_seconds", retryAfter)
		httpx.Error(w, r, http.StatusTooManyRequests, "RATE_LIMITED",
			"发送过于频繁，请"+strconv.Itoa(retryAfter)+"秒后再试")
		return
	}

	code, err := s.auth.SendLoginCode(r.Context(), mobile)
	if err != nil {
		s.log.ErrorContext(r.Context(), "send login code", "mobile", maskMobile(mobile), "err", err)
		httpx.JSON(w, http.StatusOK, &biz.SendCodeResp{Success: false, ErrorMsg: "发送失败"})
		return
	}
	s.log.InfoContext(r.Context(), "sms login code (dev)", "mobile", maskMobile(mobile), "code", code)
	httpx.JSON(w, http.StatusOK, &biz.SendCodeResp{Success: true})
}

// login 成功返回裸 LoginBean；失败返回统一 JSON 错误体（参数 400 / 凭证 401）。
func (s *Service) login(w http.ResponseWriter, r *http.Request) {
	var body loginRequest
	if err := httpx.ReadJSON(r, &body); err != nil {
		if errors.Is(err, httpx.ErrBodyTooLarge) {
			s.fail(w, r, err)
			return
		}
		httpx.Error(w, r, http.StatusBadRequest, "BAD_REQUEST", "请求格式错误")
		return
	}
	mobile := strings.TrimSpace(body.UA)
	if !mobileRe.MatchString(mobile) {
		httpx.Error(w, r, http.StatusBadRequest, "BAD_REQUEST", "手机号格式不正确")
		return
	}
	if s.needCaptcha(r, mobile, body.Captcha) {
		loginTotal.WithLabelValues("captcha").Inc()
		httpx.JSON(w, http.StatusOK, &biz.LoginBean{NeedCaptcha: true, User: nil})
		return
	}

	bean, token, err := s.auth.Login(r.Context(), mobile, body.PW, body.Code)
	if err != nil {
		loginTotal.WithLabelValues("failure").Inc()
		observability.RecordError(r.Context(), err)
		s.log.WarnContext(r.Context(), "login failed", "mobile", maskMobile(mobile), "err", err)
		s.limit.bumpFail(mobile, loginFailWindow)
		if errors.Is(err, biz.ErrBadRequest) {
			httpx.Error(w, r, http.StatusBadRequest, "BAD_REQUEST", "参数错误")
			return
		}
		httpx.Error(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "登录失败")
		return
	}
	s.limit.resetFail(mobile)
	loginTotal.WithLabelValues("success").Inc()

	_, _, _, cookieSecure := s.authSettings()
	session.DeviceID(w, r)
	session.SetCookie(w, token, cookieSecure)
	var userID int64
	if bean.User != nil {
		userID = bean.User.UserID
	}
	s.log.InfoContext(r.Context(), "login succeeded", "mobile", maskMobile(mobile), "user_id", userID)
	httpx.JSON(w, http.StatusOK, bean)
}

// needCaptcha 连续失败达到配置阈值后要求图形码；带码时校验 Cookie 绑定的验证码。
func (s *Service) needCaptcha(r *http.Request, mobile, code string) bool {
	if code != "" {
		return !s.verifyCaptcha(r, code)
	}
	_, _, threshold, _ := s.authSettings()
	return s.limit.failCount(mobile, loginFailWindow) >= threshold
}

func (s *Service) verifyCaptcha(r *http.Request, code string) bool {
	if r == nil {
		return false
	}
	c, err := r.Cookie("psCaptcha")
	if err != nil || c.Value == "" {
		return false
	}
	return s.auth.VerifyCaptcha(r.Context(), c.Value, code)
}

// authSettings 读取认证风控配置；零值（如单测构造）时回落到安全默认值。
func (s *Service) authSettings() (cooldown time.Duration, ipLimit, captchaThreshold int, cookieSecure bool) {
	cfg := s.authCfg
	cooldown = time.Duration(cfg.SendCodeCooldownSeconds) * time.Second
	if cooldown <= 0 {
		cooldown = 60 * time.Second
	}
	ipLimit = cfg.SendCodeIPLimitPerMinute
	if ipLimit <= 0 {
		ipLimit = 10
	}
	captchaThreshold = cfg.LoginFailCaptchaThreshold
	if captchaThreshold <= 0 {
		captchaThreshold = 3
	}
	return cooldown, ipLimit, captchaThreshold, cfg.CookieSecureEnabled()
}

const loginFailWindow = 10 * time.Minute

// authLimiter 是发码与登录失败计数的进程内限流器（单机语义）。
// 说明：冷却时间与阈值需按配置动态调整，且现有 Redis 端口未暴露可配置窗口，
// 因此采用带过期的进程内计数；多实例部署时各实例独立计数。
type authLimiter struct {
	mu   sync.Mutex
	send map[string]time.Time
	ip   map[string][]time.Time
	fail map[string]failEntry
}

type failEntry struct {
	count int
	until time.Time
}

func newAuthLimiter() *authLimiter {
	return &authLimiter{
		send: map[string]time.Time{},
		ip:   map[string][]time.Time{},
		fail: map[string]failEntry{},
	}
}

// allowSend 按手机号冷却与 IP 每分钟阈值双维度放行；返回是否放行、命中维度与建议重试秒数。
func (l *authLimiter) allowSend(mobile, ip string, cooldown time.Duration, ipLimit int) (bool, string, int) {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sweep(now)

	if until, ok := l.send[mobile]; ok && now.Before(until) {
		return false, "mobile", secondsUntil(until, now)
	}

	times := l.ip[ip]
	cutoff := now.Add(-time.Minute)
	kept := times[:0]
	for _, t := range times {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if ipLimit > 0 && len(kept) >= ipLimit {
		l.ip[ip] = kept
		return false, "ip", secondsUntil(kept[0].Add(time.Minute), now)
	}

	l.ip[ip] = append(kept, now)
	l.send[mobile] = now.Add(cooldown)
	return true, "", 0
}

// failCount 返回窗口内连续登录失败次数。
func (l *authLimiter) failCount(mobile string, window time.Duration) int {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.fail[mobile]
	if !ok || now.After(e.until) {
		delete(l.fail, mobile)
		return 0
	}
	return e.count
}

// bumpFail 记录一次登录失败并刷新统计窗口。
func (l *authLimiter) bumpFail(mobile string, window time.Duration) {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sweep(now)
	e, ok := l.fail[mobile]
	if !ok || now.After(e.until) {
		e = failEntry{}
	}
	e.count++
	e.until = now.Add(window)
	l.fail[mobile] = e
}

// resetFail 清空登录失败计数。
func (l *authLimiter) resetFail(mobile string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.fail, mobile)
}

// sweep 惰性清理过期条目，避免进程内 map 无界增长。
func (l *authLimiter) sweep(now time.Time) {
	if len(l.send) > 1024 {
		for k, until := range l.send {
			if now.After(until) {
				delete(l.send, k)
			}
		}
	}
	if len(l.ip) > 1024 {
		cutoff := now.Add(-time.Minute)
		for k, times := range l.ip {
			if len(times) == 0 || times[len(times)-1].Before(cutoff) {
				delete(l.ip, k)
			}
		}
	}
	if len(l.fail) > 1024 {
		for k, e := range l.fail {
			if now.After(e.until) {
				delete(l.fail, k)
			}
		}
	}
}

// clientIP 取连接对端 IP（不信任可伪造的转发头）。
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// maskMobile 对手机号中间四位做脱敏，避免日志泄露。
func maskMobile(mobile string) string {
	if len(mobile) < 7 {
		return mobile
	}
	return mobile[:3] + "****" + mobile[len(mobile)-4:]
}

// secondsUntil 返回距今剩余秒数（向上取整，至少 1 秒）。
func secondsUntil(t, now time.Time) int {
	d := t.Sub(now)
	if d <= 0 {
		return 0
	}
	return int((d + time.Second - 1) / time.Second)
}
