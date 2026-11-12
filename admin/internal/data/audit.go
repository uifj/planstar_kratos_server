package data

import (
	"context"
	"database/sql"
	"strings"

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

// ListAuditLogs pages the audit trail newest first.
func (r *auditRepo) ListAuditLogs(ctx context.Context, q biz.AuditQuery) ([]*biz.AuditLog, error) {
	query := `SELECT id, admin_id, actor, action, target, result, COALESCE(detail,''),
		       COALESCE(remote_addr,''), created_at
		  FROM planstar_admin.admin_audit_log`
	where, args := make([]string, 0, 3), make([]any, 0, 5)
	if q.Keyword != "" {
		like := "%" + q.Keyword + "%"
		where = append(where, "(actor LIKE ? OR action LIKE ? OR target LIKE ?)")
		args = append(args, like, like, like)
	}
	if q.Action != "" {
		where = append(where, "action = ?")
		args = append(args, q.Action)
	}
	if q.Result != "" {
		where = append(where, "result = ?")
		args = append(args, q.Result)
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY id DESC LIMIT ? OFFSET ?"
	args = append(args, q.Limit, q.Offset)

	rows, err := r.data.readDB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	logs := make([]*biz.AuditLog, 0, q.Limit)
	for rows.Next() {
		log := &biz.AuditLog{}
		var adminID sql.NullString
		if err := rows.Scan(&log.ID, &adminID, &log.Actor, &log.Action, &log.Target,
			&log.Result, &log.Detail, &log.RemoteAddr, &log.CreatedAt); err != nil {
			return nil, err
		}
		log.AdminID = adminID.String
		logs = append(logs, log)
	}
	return logs, rows.Err()
}
