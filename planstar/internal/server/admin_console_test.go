package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAdminConsoleServesReportsAndModerationUI(t *testing.T) {
	rec := httptest.NewRecorder()
	adminConsole(rec, httptest.NewRequest(http.MethodGet, "/admin/console", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("content-type = %q, want text/html", ct)
	}
	body := rec.Body.String()
	for _, marker := range []string{
		"举报处理台",
		"企划审核",
		"/admin/reports",
		"/admin/projects/",
		"/review",
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("console html missing marker %q", marker)
		}
	}
}
