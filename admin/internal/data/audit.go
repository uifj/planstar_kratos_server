package data

import (
	"context"

	"github.com/google/uuid"

	"planstar/admin/internal/biz"
)

type auditRepo struct {
	data *Data
}

// NewAuditRepo creates the admin_audit_log repository.
func NewAuditRepo(data *Data) biz.AuditRepo {
	return &auditRepo{data: data}
}

// Record inserts one audit row. detail is stored as text; callers never pass
// passwords or tokens.
func (r *auditRepo) Record(ctx context.Context, entry *biz.AuditEntry) error {
	var adminID any
	if entry.AdminID != uuid.Nil {
		adminID = entry.AdminID.String()
	}
	var detail any
	if entry.Detail != "" {
		detail = entry.Detail
	}
	_, err := r.data.readDB.ExecContext(ctx,
		`INSERT INTO planstar_admin.admin_audit_log
		   (admin_id, actor, action, target, result, detail, remote_addr)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		adminID, entry.Actor, entry.Action, entry.Target, entry.Result, detail, entry.RemoteAddr)
	return err
}
