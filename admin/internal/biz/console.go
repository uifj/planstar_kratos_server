package biz

import (
	"context"
	"strings"
	"time"
)

// maxConsolePageSize caps every console list query. The service layer applies
// the same cap so a crafted page_size cannot bypass it.
const maxConsolePageSize = 200

// PlanstarUser is a business user (planstar_user.user_info).
type PlanstarUser struct {
	ID        int64
	Name      string
	Mobile    string
	Avatar    string
	Status    int32
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Project is a catalog project (planstar_catalog.project).
type Project struct {
	ID                  int64
	Title               string
	Subtitle            string
	CategoryID          int64
	CategoryName        string
	PublishStatus       int32
	FundingStatus       string
	GoalAmount          int64
	AchievementPermille int32
	SupporterCount      int32
	CreatorName         string
	CreatedAt           time.Time
}

// Order is a unified order (planstar_order.order_info).
type Order struct {
	ID        int64
	OrderNo   string
	UserID    int64
	Amount    int64
	Status    int32
	CreatedAt time.Time
}

// Report is a moderation report (planstar_catalog.project_report).
type Report struct {
	ID          int64
	TargetType  string
	TargetID    int64
	ProjectID   int64
	UserID      int64
	ReasonCode  string
	Description string
	Evidence    []string
	Status      string
	HandleNote  string
	HandleTime  *time.Time
	CreatedAt   time.Time
}

// BlacklistHit is one blacklist rule match returned by the ops service.
type BlacklistHit struct {
	Rule    string
	Keyword string
}

// ReviewResult is the outcome of a project review performed by the ops service.
type ReviewResult struct {
	Project Project
	Hits    []BlacklistHit
	Blocked bool
}

// UserQuery filters and pages the user list. A nil Status means "all".
type UserQuery struct {
	Keyword string
	Status  *int32
	Offset  int
	Limit   int
}

// ProjectQuery filters and pages the project list. A nil Status means "all".
type ProjectQuery struct {
	Keyword string
	Status  *int32
	Offset  int
	Limit   int
}

// OrderQuery filters and pages the order list. A nil Status means "all".
type OrderQuery struct {
	Keyword string
	Status  *int32
	Offset  int
	Limit   int
}

// ReportQuery filters and pages the report list. An empty Status means "all".
type ReportQuery struct {
	Keyword string
	Status  string
	Offset  int
	Limit   int
}

// ConsoleRepo is the read-mostly storage port for the console. The only write
// is the user account status update.
type ConsoleRepo interface {
	ListUsers(context.Context, UserQuery) ([]*PlanstarUser, error)
	UpdateUserStatus(context.Context, int64, int32) (*PlanstarUser, error)
	ListProjects(context.Context, ProjectQuery) ([]*Project, error)
	ListOrders(context.Context, OrderQuery) ([]*Order, error)
	ListReports(context.Context, ReportQuery) ([]*Report, error)
}

// OpsClient forwards moderation decisions to the business service. The
// blacklist itself is owned by that service; this client never reimplements it.
type OpsClient interface {
	ReviewProject(ctx context.Context, id int64, decision string) (*ReviewResult, error)
	HandleReport(ctx context.Context, id int64, status, note string) (*Report, error)
}

// ConsoleUsecase implements the console business operations.
type ConsoleUsecase struct {
	repo ConsoleRepo
	ops  OpsClient
}

// NewConsoleUsecase creates a console usecase.
func NewConsoleUsecase(repo ConsoleRepo, ops OpsClient) *ConsoleUsecase {
	return &ConsoleUsecase{repo: repo, ops: ops}
}

// ListUsers lists business users.
func (uc *ConsoleUsecase) ListUsers(ctx context.Context, q UserQuery) ([]*PlanstarUser, error) {
	q.Keyword = strings.TrimSpace(q.Keyword)
	if q.Status != nil && *q.Status < 0 {
		return nil, ErrInvalidStatus
	}
	normalizeQuery(&q.Keyword, &q.Offset, &q.Limit)
	return uc.repo.ListUsers(ctx, q)
}

// UpdateUserStatus updates the account status of one business user.
func (uc *ConsoleUsecase) UpdateUserStatus(ctx context.Context, id int64, status int32) (*PlanstarUser, error) {
	if id <= 0 || status < 0 {
		return nil, ErrInvalidStatus
	}
	return uc.repo.UpdateUserStatus(ctx, id, status)
}

// ListProjects lists projects for moderation.
func (uc *ConsoleUsecase) ListProjects(ctx context.Context, q ProjectQuery) ([]*Project, error) {
	q.Keyword = strings.TrimSpace(q.Keyword)
	if q.Status != nil && (*q.Status < 0 || *q.Status > 3) {
		return nil, ErrInvalidStatus
	}
	normalizeQuery(&q.Keyword, &q.Offset, &q.Limit)
	return uc.repo.ListProjects(ctx, q)
}

// ListOrders lists unified orders.
func (uc *ConsoleUsecase) ListOrders(ctx context.Context, q OrderQuery) ([]*Order, error) {
	q.Keyword = strings.TrimSpace(q.Keyword)
	if q.Status != nil && *q.Status < 0 {
		return nil, ErrInvalidStatus
	}
	normalizeQuery(&q.Keyword, &q.Offset, &q.Limit)
	return uc.repo.ListOrders(ctx, q)
}

// ListReports lists moderation reports.
func (uc *ConsoleUsecase) ListReports(ctx context.Context, q ReportQuery) ([]*Report, error) {
	q.Status = strings.TrimSpace(q.Status)
	switch q.Status {
	case "", "pending", "accepted", "rejected":
	default:
		return nil, ErrInvalidStatus
	}
	normalizeQuery(&q.Keyword, &q.Offset, &q.Limit)
	return uc.repo.ListReports(ctx, q)
}

// ReviewProject approves or rejects a project through the ops service.
func (uc *ConsoleUsecase) ReviewProject(ctx context.Context, id int64, decision string) (*ReviewResult, error) {
	if id <= 0 {
		return nil, ErrProjectNotFound
	}
	decision = strings.TrimSpace(decision)
	if decision != "approve" && decision != "reject" {
		return nil, ErrInvalidDecision
	}
	return uc.ops.ReviewProject(ctx, id, decision)
}

// HandleReport accepts or rejects a report through the ops service.
func (uc *ConsoleUsecase) HandleReport(ctx context.Context, id int64, status, note string) (*Report, error) {
	if id <= 0 {
		return nil, ErrReportNotFound
	}
	status = strings.TrimSpace(status)
	if status != "accepted" && status != "rejected" {
		return nil, ErrInvalidStatus
	}
	return uc.ops.HandleReport(ctx, id, status, strings.TrimSpace(note))
}

// normalizeQuery applies the shared keyword trim, offset floor and page limit.
func normalizeQuery(keyword *string, offset, limit *int) {
	*keyword = strings.TrimSpace(*keyword)
	if *offset < 0 {
		*offset = 0
	}
	if *limit <= 0 {
		*limit = 20
	}
	if *limit > maxConsolePageSize {
		*limit = maxConsolePageSize
	}
}
