package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"planstar/internal/biz"
	"planstar/internal/platform/conf"
)

func okSet(context.Context, int64, int32) error { return nil }

func newStatusRequest(t *testing.T, id, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/admin/feeds/"+id+"/status", strings.NewReader(body))
	req.SetPathValue("id", id)
	return req
}

func TestOpsStatusHandlerSuccessAudits(t *testing.T) {
	auth, logs := newTestOpsAuth(t, conf.Ops{Token: "tok"})
	h := opsStatusHandler(auth, "feed.status", okSet)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, newStatusRequest(t, "1", `{"status":0}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"id":1`) || !strings.Contains(rec.Body.String(), `"status":0`) {
		t.Fatalf("body = %s, want id/status echo", rec.Body.String())
	}
	if !strings.Contains(logs.String(), "feed.status") || !strings.Contains(logs.String(), "success") {
		t.Fatalf("audit log missing feed.status success: %s", logs.String())
	}
}

func TestOpsStatusHandlerValidationAndErrors(t *testing.T) {
	cases := []struct {
		name     string
		id       string
		body     string
		setErr   error
		wantCode int
	}{
		{"bad path id", "abc", `{"status":0}`, nil, http.StatusBadRequest},
		{"zero id", "0", `{"status":0}`, nil, http.StatusBadRequest},
		{"malformed body", "1", `{`, nil, http.StatusBadRequest},
		{"invalid status", "1", `{"status":2}`, biz.ErrBadRequest, http.StatusBadRequest},
		{"missing row", "1", `{"status":0}`, biz.ErrNotFound, http.StatusNotFound},
		{"internal", "1", `{"status":0}`, errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			auth, _ := newTestOpsAuth(t, conf.Ops{Token: "tok"})
			set := func(context.Context, int64, int32) error { return tc.setErr }
			h := opsStatusHandler(auth, "feed.status", set)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, newStatusRequest(t, tc.id, tc.body))
			if rec.Code != tc.wantCode {
				t.Fatalf("code = %d, want %d", rec.Code, tc.wantCode)
			}
		})
	}
}

func TestOpsStatusHandlerAuthFailClosedAndBearer(t *testing.T) {
	t.Run("no token configured rejects", func(t *testing.T) {
		auth, _ := newTestOpsAuth(t, conf.Ops{})
		h := auth.Wrap("admin", opsStatusHandler(auth, "feed.status", okSet))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, newStatusRequest(t, "1", `{"status":0}`))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("code = %d, want 401 (fail-closed)", rec.Code)
		}
	})
	t.Run("wrong bearer rejects", func(t *testing.T) {
		auth, _ := newTestOpsAuth(t, conf.Ops{Token: "tok"})
		h := auth.Wrap("admin", opsStatusHandler(auth, "feed.status", okSet))
		req := newStatusRequest(t, "1", `{"status":0}`)
		req.Header.Set("Authorization", "Bearer wrong")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("code = %d, want 401", rec.Code)
		}
	})
	t.Run("correct bearer passes", func(t *testing.T) {
		auth, _ := newTestOpsAuth(t, conf.Ops{Token: "tok"})
		h := auth.Wrap("admin", opsStatusHandler(auth, "feed.status", okSet))
		req := newStatusRequest(t, "1", `{"status":0}`)
		req.Header.Set("Authorization", "Bearer tok")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("code = %d, want 200", rec.Code)
		}
	})
}

func TestOpsResultMapping(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{biz.ErrBadRequest, "bad_request"},
		{biz.ErrNotFound, "not_found"},
		{biz.ErrRefundNotAllowed, "not_allowed"},
		{errors.New("boom"), "error"},
	}
	for _, tc := range cases {
		if got := opsResult(tc.err); got != tc.want {
			t.Fatalf("opsResult(%v) = %q, want %q", tc.err, got, tc.want)
		}
	}
}

func TestWriteAdminErrorMapping(t *testing.T) {
	cases := []struct {
		err      error
		wantCode int
		wantBody string
	}{
		{biz.ErrBadRequest, http.StatusBadRequest, "BAD_REQUEST"},
		{biz.ErrRefundNotAllowed, http.StatusBadRequest, "REFUND_NOT_ALLOWED"},
		{biz.ErrNotFound, http.StatusNotFound, "NOT_FOUND"},
		{errors.New("boom"), http.StatusInternalServerError, "INTERNAL"},
	}
	for _, tc := range cases {
		rec := httptest.NewRecorder()
		writeAdminError(rec, tc.err)
		if rec.Code != tc.wantCode || !strings.Contains(rec.Body.String(), tc.wantBody) {
			t.Fatalf("err %v -> %d %s, want %d %s", tc.err, rec.Code, rec.Body.String(), tc.wantCode, tc.wantBody)
		}
	}
}
