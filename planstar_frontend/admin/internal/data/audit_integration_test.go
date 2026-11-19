//go:build integration

package data

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"planstar/admin/internal/biz"
	"planstar/admin/internal/testutil"
)

const integrationAuditAction = "integration.audit"

func TestAuditRecordAndListIntegration(t *testing.T) {
	db := testutil.MySQL(t)
	repo := NewAuditRepo(&Data{readDB: db})
	ctx := context.Background()
	testutil.Cleanup(t, db, `DELETE FROM planstar_admin.admin_audit_log WHERE action=?`, integrationAuditAction)

	adminID := uuid.New()
	entry := &biz.AuditEntry{
		AdminID:    adminID,
		Actor:      "it-actor",
		Action:     integrationAuditAction,
		Target:     "999101",
		Result:     "success",
		Detail:     "approve",
		RemoteAddr: "127.0.0.1",
	}
	if err := repo.Record(ctx, entry); err != nil {
		t.Fatalf("Record: %v", err)
	}
	// A second entry with nil admin and empty detail exercises the NULL paths.
	if err := repo.Record(ctx, &biz.AuditEntry{
		Actor:  "it-actor",
		Action: integrationAuditAction,
		Target: "999102",
		Result: "failure",
	}); err != nil {
		t.Fatalf("Record null fields: %v", err)
	}

	t.Run("list filters by action and keyword", func(t *testing.T) {
		logs, err := repo.ListAuditLogs(ctx, biz.AuditQuery{
			Action:  integrationAuditAction,
			Keyword: "999101",
			Limit:   10,
		})
		if err != nil {
			t.Fatalf("ListAuditLogs: %v", err)
		}
		if len(logs) != 1 {
			t.Fatalf("logs = %d, want 1", len(logs))
		}
		got := logs[0]
		if got.AdminID != adminID.String() || got.Actor != "it-actor" ||
			got.Target != "999101" || got.Result != "success" || got.Detail != "approve" {
			t.Fatalf("log = %+v, want recorded entry", got)
		}
	})

	t.Run("list filters by result and pages", func(t *testing.T) {
		failures, err := repo.ListAuditLogs(ctx, biz.AuditQuery{
			Action: integrationAuditAction,
			Result: "failure",
			Limit:  10,
		})
		if err != nil {
			t.Fatalf("ListAuditLogs failure: %v", err)
		}
		if len(failures) != 1 || failures[0].Target != "999102" {
			t.Fatalf("failures = %+v, want the 999102 entry", failures)
		}
		if failures[0].AdminID != "" || failures[0].Detail != "" {
			t.Fatalf("null fields = %q/%q, want empty strings", failures[0].AdminID, failures[0].Detail)
		}
		// Offset past the only failure row yields an empty page, not an error.
		empty, err := repo.ListAuditLogs(ctx, biz.AuditQuery{
			Action: integrationAuditAction,
			Result: "failure",
			Limit:  10,
			Offset: 10,
		})
		if err != nil || len(empty) != 0 {
			t.Fatalf("offset page = %d err %v, want 0/nil", len(empty), err)
		}
	})
}
