package session

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"
)

// CookieKey 是会话 Cookie 名（契约：Cookie: psSession=…）。
const CookieKey = "psSession"

// SessionTTL 会话默认有效期。
const SessionTTL = 30 * 24 * time.Hour

// NewToken 生成随机会话令牌（64 hex）。
func NewToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

// SetCookie 下发 psSession：同时写 Set-Cookie 与响应头（客户端两处都读）。
// secure 可选，缺省为 false；生产环境应传入 true 以启用 Secure。
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
	w.Header().Set(CookieKey, token)
}

// FromRequest 从 Cookie 中读取 psSession。
func FromRequest(r *http.Request) string {
	if c, err := r.Cookie(CookieKey); err == nil && c.Value != "" {
		return c.Value
	}
	return r.Header.Get(CookieKey)
}

// DeviceID 读取 dvid；缺省时下发一个新设备标识（Set-Cookie + 响应头）。
func DeviceID(w http.ResponseWriter, r *http.Request) string {
	dvid := r.Header.Get("dvid")
	if dvid == "" {
		if c, err := r.Cookie("dvid"); err == nil && c.Value != "" {
			dvid = c.Value
		}
	}
	if dvid == "" {
		dvid = NewToken()
		http.SetCookie(w, &http.Cookie{Name: "dvid", Value: dvid, Path: "/", MaxAge: 365 * 24 * 3600})
		w.Header().Set("dvid", dvid)
	}
	return dvid
}
