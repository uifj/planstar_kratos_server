package service

import (
	"errors"
	"net/http"
	"regexp"
	"strings"

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

// sendLoginCode 契约：恒 200 + {success, needCaptcha, errorMsg}。
func (s *Service) sendLoginCode(w http.ResponseWriter, r *http.Request) {
	var body sendCodeRequest
	if err := httpx.ReadJSON(r, &body); err != nil {
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
	if !s.auth.AllowSendCode(r.Context(), mobile) {
		httpx.JSON(w, http.StatusOK, &biz.SendCodeResp{Success: false, NeedCaptcha: true, ErrorMsg: "发送过于频繁，请稍后再试"})
		return
	}

	code, err := s.auth.SendLoginCode(r.Context(), mobile)
	if err != nil {
		s.log.ErrorContext(r.Context(), "send login code", "err", err)
		httpx.JSON(w, http.StatusOK, &biz.SendCodeResp{Success: false, ErrorMsg: "发送失败"})
		return
	}
	s.log.InfoContext(r.Context(), "sms login code (dev)", "mobile", mobile, "code", code)
	httpx.JSON(w, http.StatusOK, &biz.SendCodeResp{Success: true})
}

// login 契约：成功裸 LoginBean；失败 5xx + text/plain。
func (s *Service) login(w http.ResponseWriter, r *http.Request) {
	var body loginRequest
	if err := httpx.ReadJSON(r, &body); err != nil {
		httpx.Plain(w, http.StatusInternalServerError, "请求格式错误")
		return
	}
	mobile := strings.TrimSpace(body.UA)
	if !mobileRe.MatchString(mobile) {
		httpx.Plain(w, http.StatusInternalServerError, "手机号格式不正确")
		return
	}
	if s.needCaptcha(r, mobile, body.Captcha) {
		httpx.JSON(w, http.StatusOK, &biz.LoginBean{NeedCaptcha: true, User: nil})
		return
	}

	bean, token, err := s.auth.Login(r.Context(), mobile, body.PW, body.Code)
	if err != nil {
		observability.RecordError(r.Context(), err)
		s.log.WarnContext(r.Context(), "login failed", "mobile", mobile, "err", err)
		s.auth.BumpLoginFail(r.Context(), mobile)
		if errors.Is(err, biz.ErrBadRequest) {
			httpx.Plain(w, http.StatusInternalServerError, "参数错误")
			return
		}
		httpx.Plain(w, http.StatusInternalServerError, "登录失败")
		return
	}
	s.auth.ResetLoginFail(r.Context(), mobile)

	session.DeviceID(w, r)
	session.SetCookie(w, token)
	httpx.JSON(w, http.StatusOK, bean)
}

// needCaptcha 连续失败 3 次后要求图形码；带码时校验 Cookie 绑定的验证码。
func (s *Service) needCaptcha(r *http.Request, mobile, code string) bool {
	if code != "" {
		return !s.verifyCaptcha(r, code)
	}
	return s.auth.NeedsCaptcha(r.Context(), mobile)
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
