package session

import (
	"net/http"

	"planstar/internal/platform/pkg/token"
)

// CookieKey 是会话 Cookie 名（契约：Cookie: psSession=…）。
const CookieKey = "psSession"

// DeviceCookieKey 是设备标识 Cookie 名。
const DeviceCookieKey = "dvid"

// SessionTTL 会话默认有效期（纯算法常量来自 token 包）。
const SessionTTL = token.SessionTTL

// DeviceTTL 设备标识有效期（7 天，原实现为 365 天且无任何安全属性）。
const DeviceTTL = token.DeviceTTL

// NewToken 兼容包装。令牌生成算法已下沉到无 net/http 依赖的 token 包，
// 领域层（biz）应直接调用 token.New，避免传递依赖 HTTP 适配器。
func NewToken() (string, error) { return token.New() }

// SetCookie 下发 psSession。
//
// 安全要求（OWASP Session Management / 《阿里巴巴Java开发手册（黄山版）》安全规约）：
// 会话令牌只能通过 HttpOnly + Secure + SameSite Cookie 传递，
// 禁止再写入 JS 可读的响应头，否则 HttpOnly 形同虚设。
func SetCookie(w http.ResponseWriter, token string, secure ...bool) {
	c := &http.Cookie{
		Name:     CookieKey,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   len(secure) > 0 && secure[0],
		MaxAge:   int(SessionTTL.Seconds()),
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, c)
}

// FromRequest 从 Cookie 中读取 psSession。
// 兼容旧客户端的自定义请求头，但响应侧已不再下发该头。
func FromRequest(r *http.Request) string {
	if c, err := r.Cookie(CookieKey); err == nil && c.Value != "" {
		return c.Value
	}
	return r.Header.Get(CookieKey)
}

// DeviceID 读取 dvid；缺省时下发一个新设备标识。保留旧签名供测试与调用方使用，
// 默认不启用 Secure（本地 HTTP 联调）。
func DeviceID(w http.ResponseWriter, r *http.Request) string {
	return DeviceIDWithOptions(w, r, false)
}

// DeviceIDWithOptions 读取 dvid；secure 由 server.env 推导，生产为 true。
// 设备 Cookie 同样强制 HttpOnly + SameSite，并缩短有效期。
func DeviceIDWithOptions(w http.ResponseWriter, r *http.Request, secure bool) string {
	dvid := r.Header.Get(DeviceCookieKey)
	if dvid == "" {
		if c, err := r.Cookie(DeviceCookieKey); err == nil && c.Value != "" {
			dvid = c.Value
		}
	}
	if dvid == "" {
		token, err := NewToken()
		if err != nil {
			return ""
		}
		dvid = token
		http.SetCookie(w, &http.Cookie{
			Name:     DeviceCookieKey,
			Value:    dvid,
			Path:     "/",
			HttpOnly: true,
			Secure:   secure,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   int(DeviceTTL.Seconds()),
		})
		// dvid 不是会话凭证，仍通过自定义头下发给 Flutter 客户端做设备标识。
		w.Header().Set(DeviceCookieKey, dvid)
	}
	return dvid
}
