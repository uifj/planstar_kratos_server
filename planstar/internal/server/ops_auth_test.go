package server

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"planstar/internal/conf"
	"planstar/internal/observability"
)

func newTestOpsAuth(t *testing.T, c conf.Ops) (*OpsAuth, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	auth, err := NewOpsAuth(c, logger, observability.NewMetrics("v-test", "dev"))
	if err != nil {
		t.Fatalf("NewOpsAuth: %v", err)
	}
	return auth, &buf
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}

func TestOpsAuthRequiresBearerToken(t *testing.T) {
	const token = "s3cret-token"
	auth, _ := newTestOpsAuth(t, conf.Ops{Token: token})
	h := auth.Wrap("metrics", okHandler())

	cases := []struct {
		name   string
		header string
	}{
		{name: "missing", header: ""},
		{name: "wrong", header: "Bearer wrong-token"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want 401", rec.Code)
			}
			if got := rec.Header().Get("WWW-Authenticate"); got != "Bearer" {
				t.Fatalf("WWW-Authenticate = %q, want Bearer", got)
			}
			if !strings.Contains(rec.Body.String(), "UNAUTHORIZED") {
				t.Fatalf("body = %q, want UNAUTHORIZED", rec.Body.String())
			}
		})
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || rec.Body.String() != "ok" {
		t.Fatalf("status = %d body = %q, want 200/ok", rec.Code, rec.Body.String())
	}
}

func TestOpsAuthCIDRAllowlist(t *testing.T) {
	auth, _ := newTestOpsAuth(t, conf.Ops{
		Token:        "s3cret-token",
		AllowedCIDRs: []string{"192.168.0.0/16"},
	})
	h := auth.Wrap("admin", okHandler())

	forbidden := httptest.NewRequest(http.MethodGet, "/admin/reports", nil)
	forbidden.RemoteAddr = "10.1.2.3:4567"
	forbidden.Header.Set("Authorization", "Bearer s3cret-token")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, forbidden)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "FORBIDDEN") {
		t.Fatalf("body = %q, want FORBIDDEN", rec.Body.String())
	}

	allowed := httptest.NewRequest(http.MethodGet, "/admin/reports", nil)
	allowed.RemoteAddr = "192.168.1.9:4567"
	allowed.Header.Set("Authorization", "Bearer s3cret-token")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, allowed)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestOpsProbesRemainAnonymous(t *testing.T) {
	ready := newTestReadiness(t, errors.New("mysql down"))
	live, readyz := NewProbeHandlers(ready, "v-test", nil)

	rec := httptest.NewRecorder()
	live.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("healthz status = %d, want 200", rec.Code)
	}

	rec = httptest.NewRecorder()
	readyz.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz status = %d, want 503", rec.Code)
	}
}

func TestOpsAuthAuditLogDoesNotLeakToken(t *testing.T) {
	const token = "audit-secret-token"
	auth, buf := newTestOpsAuth(t, conf.Ops{Token: token})
	h := auth.Wrap("admin", okHandler())

	req := httptest.NewRequest(http.MethodGet, "/admin/reports", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	logs := buf.String()
	if !strings.Contains(logs, "audit=true") {
		t.Fatalf("audit log missing audit=true: %q", logs)
	}
	if !strings.Contains(logs, "action=ops.admin.access") {
		t.Fatalf("audit log missing action: %q", logs)
	}
	if strings.Contains(logs, token) {
		t.Fatalf("audit log leaked token: %q", logs)
	}
}

func TestOpsAuthValidate(t *testing.T) {
	empty, _ := newTestOpsAuth(t, conf.Ops{})
	if err := empty.Validate("prod"); err == nil || !strings.Contains(err.Error(), "PLANSTAR_OPS_TOKEN") {
		t.Fatalf("Validate(prod) = %v, want PLANSTAR_OPS_TOKEN error", err)
	}
	if err := empty.Validate("dev"); err != nil {
		t.Fatalf("Validate(dev) = %v, want nil", err)
	}

	withToken, _ := newTestOpsAuth(t, conf.Ops{Token: "configured"})
	if err := withToken.Validate("prod"); err != nil {
		t.Fatalf("Validate(prod) = %v, want nil", err)
	}
}
