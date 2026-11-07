package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"planstar/admin/internal/conf"
)

type fakeReadiness struct {
	err error
}

func (f fakeReadiness) Ready(context.Context) error { return f.err }

func serveOps(t *testing.T, ready ReadinessChecker, path string) *httptest.ResponseRecorder {
	t.Helper()
	srv := NewOpsServer(&conf.Server{}, ready, NewMetrics())
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

func TestOpsHealthzAlwaysOK(t *testing.T) {
	rec := serveOps(t, fakeReadiness{err: errors.New("mysql down")}, "/healthz")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestOpsReadyzOK(t *testing.T) {
	rec := serveOps(t, fakeReadiness{}, "/readyz")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %s", rec.Code, rec.Body.String())
	}
}

func TestOpsReadyzUnavailable(t *testing.T) {
	rec := serveOps(t, fakeReadiness{err: errors.New("redis not ready: connection refused")}, "/readyz")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	if body := rec.Body.String(); body == "" {
		t.Fatal("expected the failing dependency in the body")
	}
}

func TestOpsMetricsEndpoint(t *testing.T) {
	rec := serveOps(t, fakeReadiness{}, "/metrics")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestNormalizePath(t *testing.T) {
	cases := map[string]string{
		"/v1/admins/list": "/v1/admins/list",
		"/v1/admins/01920000-0000-7000-8000-000000000001": "/v1/admins/:id",
		"/v1/admins/42": "/v1/admins/:id",
		"/":             "/",
	}
	for in, want := range cases {
		if got := normalizePath(in); got != want {
			t.Errorf("normalizePath(%q) = %q, want %q", in, got, want)
		}
	}
}
