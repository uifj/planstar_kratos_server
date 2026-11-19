package session

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSetAndReadCookie(t *testing.T) {
	rec := httptest.NewRecorder()
	SetCookie(rec, "tok-123")

	// 会话令牌禁止出现在 JS 可读响应头，否则 HttpOnly 失效。
	if got := rec.Header().Get(CookieKey); got != "" {
		t.Fatalf("response header %s must not contain session token, got %q", CookieKey, got)
	}
	setCookie := rec.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, CookieKey+"=tok-123") || !strings.Contains(setCookie, "HttpOnly") {
		t.Fatalf("unexpected Set-Cookie: %q", setCookie)
	}
	if !strings.Contains(setCookie, "SameSite=Lax") {
		t.Fatalf("Set-Cookie missing SameSite: %q", setCookie)
	}

	secureRec := httptest.NewRecorder()
	SetCookie(secureRec, "tok-123", true)
	if !strings.Contains(secureRec.Header().Get("Set-Cookie"), "Secure") {
		t.Fatalf("production Set-Cookie missing Secure: %q", secureRec.Header().Get("Set-Cookie"))
	}

	req := httptest.NewRequest("GET", "/v1/user", nil)
	req.AddCookie(&http.Cookie{Name: CookieKey, Value: "tok-123"})
	if got := FromRequest(req); got != "tok-123" {
		t.Fatalf("FromRequest = %q", got)
	}

	req2 := httptest.NewRequest("GET", "/v1/user", nil)
	req2.Header.Set(CookieKey, "hdr-456")
	if got := FromRequest(req2); got != "hdr-456" {
		t.Fatalf("FromRequest header fallback = %q", got)
	}
}

func TestDeviceID(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/v1/user", nil)
	id := DeviceID(rec, req)
	if id == "" {
		t.Fatal("DeviceID should generate a value")
	}
	if rec.Header().Get("dvid") != id {
		t.Fatal("DeviceID should echo in response header")
	}
	devCookie := rec.Header().Get("Set-Cookie")
	if !strings.Contains(devCookie, "HttpOnly") || !strings.Contains(devCookie, "SameSite=Lax") {
		t.Fatalf("device cookie missing security attributes: %q", devCookie)
	}

	req2 := httptest.NewRequest("GET", "/v1/user", nil)
	req2.Header.Set("dvid", "existing")
	if got := DeviceID(httptest.NewRecorder(), req2); got != "existing" {
		t.Fatalf("DeviceID = %q, want existing", got)
	}
}
