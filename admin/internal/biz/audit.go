package biz

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

// AuditEntry is one console write recorded in planstar_admin.admin_audit_log.
type AuditEntry struct {
	AdminID    uuid.UUID
	Actor      string
	Action     string
	Target     string
	Result     string
	Detail     string
	RemoteAddr string
}

// AuditRepo persists audit entries.
type AuditRepo interface {
	Record(context.Context, *AuditEntry) error
}

// AuditUsecase records write actions without ever blocking the main flow: a
// missing audit table or a transient MySQL error is logged, not returned.
type AuditUsecase struct {
	repo   AuditRepo
	logger *slog.Logger
}

// NewAuditUsecase creates an audit usecase.
func NewAuditUsecase(repo AuditRepo, logger *slog.Logger) *AuditUsecase {
	return &AuditUsecase{repo: repo, logger: logger}
}

// Record writes one audit entry. Failures are logged only.
func (uc *AuditUsecase) Record(ctx context.Context, entry *AuditEntry) {
	if entry == nil {
		return
	}
	adminID := ""
	if entry.AdminID != uuid.Nil {
		adminID = entry.AdminID.String()
	}
	if err := uc.repo.Record(ctx, entry); err != nil {
		uc.logger.WarnContext(ctx, "console audit write failed",
			slog.Bool("audit", true),
			slog.String("admin_id", adminID),
			slog.String("actor", entry.Actor),
			slog.String("action", entry.Action),
			slog.String("target", entry.Target),
			slog.String("result", entry.Result),
			slog.String("remote_addr", entry.RemoteAddr),
			slog.Any("error", err),
		)
		return
	}
	uc.logger.InfoContext(ctx, "admin audit",
		slog.Bool("audit", true),
		slog.String("admin_id", adminID),
		slog.String("actor", entry.Actor),
		slog.String("action", entry.Action),
		slog.String("target", entry.Target),
		slog.String("result", entry.Result),
		slog.String("remote_addr", entry.RemoteAddr),
	)
}
