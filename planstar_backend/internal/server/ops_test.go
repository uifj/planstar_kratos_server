package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"planstar/internal/ops"
)

func newTestReadiness(t *testing.T, mysqlErr error) *ops.Readiness {
	t.Helper()
	ready := ops.NewReadiness([]ops.Check{
		{Name: "mysql", Critical: true, Probe: func(context.Context) error { return mysqlErr }},
		{Name: "rabbitmq", Critical: false, Probe: nil},
	}, ops.Options{
		Service:      "planstar.api",
		Version:      "v-test",
		TTL:          time.Second,
		TotalTimeout: time.Second,
	})
	ready.MarkStarted()
	return ready
}

func TestHealthzIgnoresDependencyFailure(t *testing.T) {
	ready := newTestReadiness(t, errors.New("mysql down"))
	live, _ := NewProbeHandlers(ready, "v-test", nil)

	rec := httptest.NewRecorder()
	live.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("content-type = %q, want application/json", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("cache-control = %q, want no-store", cc)
	}
	var resp ops.LiveResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if resp.Status != ops.StatusOK || resp.Version != "v-test" || resp.Service != "planstar.api" {
		t.Fatalf("live response = %+v", resp)
	}
	if resp.UptimeSeconds < 0 {
		t.Fatalf("uptime = %d", resp.UptimeSeconds)
	}
}

func TestReadyzCriticalFailureReturns503(t *testing.T) {
	ready := newTestReadiness(t, errors.New("password=secret dial failed"))
	_, readyz := NewProbeHandlers(ready, "v-test", nil)

	rec := httptest.NewRecorder()
	readyz.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("content-type = %q, want application/json", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("cache-control = %q, want no-store", cc)
	}
	var resp ops.ReadyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if resp.Ready || resp.Status != ops.StatusFail {
		t.Fatalf("ready=%v status=%q, want false/fail", resp.Ready, resp.Status)
	}
	if len(resp.Checks) != 2 || resp.Checks[0].Name != "mysql" || resp.Checks[0].Status != ops.StatusFail {
		t.Fatalf("checks = %+v", resp.Checks)
	}
	if strings.Contains(resp.Checks[0].Error, "secret") {
		t.Fatalf("错误信息未脱敏: %q", resp.Checks[0].Error)
	}
}

func TestReadyzHealthyReturns200(t *testing.T) {
	ready := newTestReadiness(t, nil)
	_, readyz := NewProbeHandlers(ready, "v-test", nil)

	rec := httptest.NewRecorder()
	readyz.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp ops.ReadyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if !resp.Ready || resp.Status != ops.StatusOK {
		t.Fatalf("ready=%v status=%q", resp.Ready, resp.Status)
	}
	if resp.Checks[1].Status != ops.StatusSkipped {
		t.Fatalf("nil probe status = %q, want skipped", resp.Checks[1].Status)
	}
}
