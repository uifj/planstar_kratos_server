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

	if got := rec.Header().Get(CookieKey); got != "tok-123" {
		t.Fatalf("response header %s = %q", CookieKey, got)
	}
	setCookie := rec.Header().Get("Set-Cookie")
	if !strings.Contains(setCookie, CookieKey+"=tok-123") || !strings.Contains(setCookie, "HttpOnly") {
		t.Fatalf("unexpected Set-Cookie: %q", setCookie)
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

	req2 := httptest.NewRequest("GET", "/v1/user", nil)
	req2.Header.Set("dvid", "existing")
	if got := DeviceID(httptest.NewRecorder(), req2); got != "existing" {
		t.Fatalf("DeviceID = %q, want existing", got)
	}
}
