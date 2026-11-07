package service

import (
	"context"
	"net"
	"strconv"
	"time"

	httpm "github.com/go-kratos/kratos/v3/transport/http"
	"go.einride.tech/aip/pagination"
	"google.golang.org/protobuf/types/known/timestamppb"

	consolev1 "planstar/admin/api/planstar/admin/v1"
	"planstar/admin/internal/biz"
	"planstar/admin/pkg/auth"
)

const (
	consoleDefaultPageSize = 20
	consoleMaxPageSize     = 200
)

// ConsoleService adapts the console HTTP contract to the business usecase.
type ConsoleService struct {
	consolev1.UnimplementedConsoleServiceServer

	uc    *biz.ConsoleUsecase
	audit *biz.AuditUsecase
}

// NewConsoleService creates the console service.
func NewConsoleService(uc *biz.ConsoleUsecase, audit *biz.AuditUsecase) *ConsoleService {
	return &ConsoleService{uc: uc, audit: audit}
}

func convertConsoleUser(user *biz.PlanstarUser) *consolev1.User {
	return &consolev1.User{
		Id:        user.ID,
		Name:      user.Name,
		Mobile:    user.Mobile,
		Avatar:    user.Avatar,
		Status:    user.Status,
		CreatedAt: timestamp(user.CreatedAt),
		UpdatedAt: timestamp(user.UpdatedAt),
	}
}

func convertConsoleProject(project *biz.Project) *consolev1.Project {
	return &consolev1.Project{
		Id:                  project.ID,
		Title:               project.Title,
		Subtitle:            project.Subtitle,
		CategoryId:          project.CategoryID,
		CategoryName:        project.CategoryName,
		PublishStatus:       project.PublishStatus,
		FundingStatus:       project.FundingStatus,
		GoalAmount:          project.GoalAmount,
		AchievementPermille: project.AchievementPermille,
		SupporterCount:      project.SupporterCount,
		CreatorName:         project.CreatorName,
		CreatedAt:           timestamp(project.CreatedAt),
	}
}

func convertConsoleOrder(order *biz.Order) *consolev1.Order {
	return &consolev1.Order{
		Id:        order.ID,
		OrderNo:   order.OrderNo,
		UserId:    order.UserID,
		Amount:    order.Amount,
		Status:    order.Status,
		CreatedAt: timestamp(order.CreatedAt),
	}
}

func convertConsoleReport(report *biz.Report) *consolev1.Report {
	out := &consolev1.Report{
		Id:          report.ID,
		TargetType:  report.TargetType,
		TargetId:    report.TargetID,
		ProjectId:   report.ProjectID,
		UserId:      report.UserID,
		ReasonCode:  report.ReasonCode,
		Description: report.Description,
		Evidence:    report.Evidence,
		Status:      report.Status,
		HandleNote:  report.HandleNote,
		CreatedAt:   timestamp(report.CreatedAt),
	}
	if report.HandleTime != nil {
		out.HandleTime = timestamp(*report.HandleTime)
	}
	return out
}

func timestamp(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

func consolePageSize(size int32) int {
	switch {
	case size <= 0:
		return consoleDefaultPageSize
	case size > consoleMaxPageSize:
		return consoleMaxPageSize
	default:
		return int(size)
	}
}

func consoleRemoteAddr(ctx context.Context) string {
	req, ok := httpm.RequestFromServerContext(ctx)
	if !ok {
		return ""
	}
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return req.RemoteAddr
	}
	return host
}

func (s *ConsoleService) viewer(ctx context.Context) (*auth.Auth, error) {
	a, ok := auth.FromContext(ctx)
	if !ok {
		return nil, auth.ErrUnauthenticated
	}
	if !a.HasViewerAccess() {
		return nil, auth.ErrPermissionDenied
	}
	return a, nil
}

func (s *ConsoleService) operator(ctx context.Context) (*auth.Auth, error) {
	a, ok := auth.FromContext(ctx)
	if !ok {
		return nil, auth.ErrUnauthenticated
	}
	if !a.HasOperatorAccess() {
		return nil, auth.ErrPermissionDenied
	}
	return a, nil
}

// audit records one write attempt. Audit failures are swallowed by the usecase.
func (s *ConsoleService) auditWrite(ctx context.Context, a *auth.Auth, action, target, result, detail string) {
	s.audit.Record(ctx, &biz.AuditEntry{
		AdminID:    a.UserID,
		Actor:      a.UserID.String(),
		Action:     action,
		Target:     target,
		Result:     result,
		Detail:     detail,
		RemoteAddr: consoleRemoteAddr(ctx),
	})
}

// ListUsers lists business users.
func (s *ConsoleService) ListUsers(ctx context.Context, req *consolev1.ListUsersRequest) (*consolev1.UserSet, error) {
	if _, err := s.viewer(ctx); err != nil {
		return nil, err
	}
	pageToken, err := pagination.ParsePageToken(req)
	if err != nil {
		return nil, err
	}
	limit := consolePageSize(req.PageSize)
	req.PageSize = int32(limit)
	users, err := s.uc.ListUsers(ctx, biz.UserQuery{
		Keyword: req.Keyword,
		Status:  req.Status,
		Offset:  int(pageToken.Offset),
		Limit:   limit,
	})
	if err != nil {
		return nil, err
	}
	set := &consolev1.UserSet{Users: make([]*consolev1.User, 0, len(users))}
	if len(users) >= limit {
		set.NextPageToken = pageToken.Next(req).String()
	}
	for _, user := range users {
		set.Users = append(set.Users, convertConsoleUser(user))
	}
	return set, nil
}

// UpdateUserStatus updates a business user's account status.
func (s *ConsoleService) UpdateUserStatus(ctx context.Context, req *consolev1.UpdateUserStatusRequest) (*consolev1.User, error) {
	a, err := s.operator(ctx)
	if err != nil {
		return nil, err
	}
	target := strconv.FormatInt(req.Id, 10)
	user, err := s.uc.UpdateUserStatus(ctx, req.Id, req.Status)
	if err != nil {
		s.auditWrite(ctx, a, "user.status", target, "failure", err.Error())
		return nil, err
	}
	s.auditWrite(ctx, a, "user.status", target, "success", "")
	return convertConsoleUser(user), nil
}

// ListProjects lists projects for moderation.
func (s *ConsoleService) ListProjects(ctx context.Context, req *consolev1.ListProjectsRequest) (*consolev1.ProjectSet, error) {
	if _, err := s.viewer(ctx); err != nil {
		return nil, err
	}
	pageToken, err := pagination.ParsePageToken(req)
	if err != nil {
		return nil, err
	}
	limit := consolePageSize(req.PageSize)
	req.PageSize = int32(limit)
	projects, err := s.uc.ListProjects(ctx, biz.ProjectQuery{
		Keyword: req.Keyword,
		Status:  req.PublishStatus,
		Offset:  int(pageToken.Offset),
		Limit:   limit,
	})
	if err != nil {
		return nil, err
	}
	set := &consolev1.ProjectSet{Projects: make([]*consolev1.Project, 0, len(projects))}
	if len(projects) >= limit {
		set.NextPageToken = pageToken.Next(req).String()
	}
	for _, project := range projects {
		set.Projects = append(set.Projects, convertConsoleProject(project))
	}
	return set, nil
}

// ReviewProject approves or rejects a project through the ops service.
func (s *ConsoleService) ReviewProject(ctx context.Context, req *consolev1.ReviewProjectRequest) (*consolev1.ReviewProjectReply, error) {
	a, err := s.operator(ctx)
	if err != nil {
		return nil, err
	}
	target := strconv.FormatInt(req.Id, 10)
	result, err := s.uc.ReviewProject(ctx, req.Id, req.Decision)
	if err != nil {
		s.auditWrite(ctx, a, "project.review", target, "failure", err.Error())
		return nil, err
	}
	s.auditWrite(ctx, a, "project.review", target, "success", req.Decision)
	hits := make([]*consolev1.BlacklistHit, 0, len(result.Hits))
	for _, hit := range result.Hits {
		hits = append(hits, &consolev1.BlacklistHit{Rule: hit.Rule, Keyword: hit.Keyword})
	}
	return &consolev1.ReviewProjectReply{
		Project: convertConsoleProject(&result.Project),
		Hits:    hits,
		Blocked: result.Blocked,
	}, nil
}

// ListOrders lists unified orders.
func (s *ConsoleService) ListOrders(ctx context.Context, req *consolev1.ListOrdersRequest) (*consolev1.OrderSet, error) {
	if _, err := s.viewer(ctx); err != nil {
		return nil, err
	}
	pageToken, err := pagination.ParsePageToken(req)
	if err != nil {
		return nil, err
	}
	limit := consolePageSize(req.PageSize)
	req.PageSize = int32(limit)
	orders, err := s.uc.ListOrders(ctx, biz.OrderQuery{
		Keyword: req.Keyword,
		Status:  req.Status,
		Offset:  int(pageToken.Offset),
		Limit:   limit,
	})
	if err != nil {
		return nil, err
	}
	set := &consolev1.OrderSet{Orders: make([]*consolev1.Order, 0, len(orders))}
	if len(orders) >= limit {
		set.NextPageToken = pageToken.Next(req).String()
	}
	for _, order := range orders {
		set.Orders = append(set.Orders, convertConsoleOrder(order))
	}
	return set, nil
}

// ListReports lists moderation reports.
func (s *ConsoleService) ListReports(ctx context.Context, req *consolev1.ListReportsRequest) (*consolev1.ReportSet, error) {
	if _, err := s.viewer(ctx); err != nil {
		return nil, err
	}
	pageToken, err := pagination.ParsePageToken(req)
	if err != nil {
		return nil, err
	}
	limit := consolePageSize(req.PageSize)
	req.PageSize = int32(limit)
	reports, err := s.uc.ListReports(ctx, biz.ReportQuery{
		Status: req.Status,
		Offset: int(pageToken.Offset),
		Limit:  limit,
	})
	if err != nil {
		return nil, err
	}
	set := &consolev1.ReportSet{Reports: make([]*consolev1.Report, 0, len(reports))}
	if len(reports) >= limit {
		set.NextPageToken = pageToken.Next(req).String()
	}
	for _, report := range reports {
		set.Reports = append(set.Reports, convertConsoleReport(report))
	}
	return set, nil
}

// HandleReport accepts or rejects a report through the ops service.
func (s *ConsoleService) HandleReport(ctx context.Context, req *consolev1.HandleReportRequest) (*consolev1.Report, error) {
	a, err := s.operator(ctx)
	if err != nil {
		return nil, err
	}
	target := strconv.FormatInt(req.Id, 10)
	report, err := s.uc.HandleReport(ctx, req.Id, req.Status, req.Note)
	if err != nil {
		s.auditWrite(ctx, a, "report.handle", target, "failure", err.Error())
		return nil, err
	}
	s.auditWrite(ctx, a, "report.handle", target, "success", req.Status)
	return convertConsoleReport(report), nil
}
