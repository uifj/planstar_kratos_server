package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	kerrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/transport"
	"github.com/google/uuid"

	consolev1 "planstar/admin/api/planstar/admin/v1"
	"planstar/admin/internal/biz"
	"planstar/admin/pkg/auth"
)

type fakeConsoleRepo struct {
	users         map[int64]*biz.PlanstarUser
	lastUserQuery biz.UserQuery
	updateCalls   int
}

func newFakeConsoleRepo() *fakeConsoleRepo {
	return &fakeConsoleRepo{users: map[int64]*biz.PlanstarUser{
		7: {ID: 7, Name: "star", Mobile: "13800000007", Status: 1, CreatedAt: time.Now()},
	}}
}

func (r *fakeConsoleRepo) ListUsers(_ context.Context, q biz.UserQuery) ([]*biz.PlanstarUser, error) {
	r.lastUserQuery = q
	return []*biz.PlanstarUser{r.users[7]}, nil
}

func (r *fakeConsoleRepo) UpdateUserStatus(_ context.Context, id int64, status int32) (*biz.PlanstarUser, error) {
	r.updateCalls++
	user, ok := r.users[id]
	if !ok {
		return nil, biz.ErrUserNotFound
	}
	user.Status = status
	return user, nil
}

func (r *fakeConsoleRepo) ListProjects(context.Context, biz.ProjectQuery) ([]*biz.Project, error) {
	return nil, nil
}

func (r *fakeConsoleRepo) ListOrders(context.Context, biz.OrderQuery) ([]*biz.Order, error) {
	return nil, nil
}

func (r *fakeConsoleRepo) ListReports(context.Context, biz.ReportQuery) ([]*biz.Report, error) {
	return nil, nil
}

type fakeOps struct {
	reviewErr    error
	reportErr    error
	reviewResult *biz.ReviewResult
	reportResult *biz.Report
	lastDecision string
	lastStatus   string
}

func (f *fakeOps) ReviewProject(_ context.Context, id int64, decision string) (*biz.ReviewResult, error) {
	f.lastDecision = decision
	if f.reviewErr != nil {
		return nil, f.reviewErr
	}
	if f.reviewResult != nil {
		return f.reviewResult, nil
	}
	return &biz.ReviewResult{Project: biz.Project{ID: id, PublishStatus: 1}}, nil
}

func (f *fakeOps) HandleReport(_ context.Context, id int64, status, note string) (*biz.Report, error) {
	f.lastStatus = status
	if f.reportErr != nil {
		return nil, f.reportErr
	}
	if f.reportResult != nil {
		return f.reportResult, nil
	}
	return &biz.Report{ID: id, Status: status, HandleNote: note}, nil
}

type fakeAuditRepo struct {
	entries []*biz.AuditEntry
	err     error
}

func (r *fakeAuditRepo) Record(_ context.Context, entry *biz.AuditEntry) error {
	copied := *entry
	r.entries = append(r.entries, &copied)
	return r.err
}

func newConsoleService() (*ConsoleService, *fakeConsoleRepo, *fakeOps, *fakeAuditRepo) {
	repo := newFakeConsoleRepo()
	ops := &fakeOps{}
	audit := &fakeAuditRepo{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewConsoleService(biz.NewConsoleUsecase(repo, ops), biz.NewAuditUsecase(audit, logger)), repo, ops, audit
}

func consoleContext(role string) context.Context {
	req, _ := http.NewRequest(http.MethodGet, "/v1/planstar/users", nil)
	req.RemoteAddr = "203.0.113.9:5555"
	tr := &fakeTransport{header: make(http.Header), request: req}
	ctx := transport.NewServerContext(context.Background(), tr)
	return auth.NewContext(ctx, &auth.Auth{UserID: uuid.New(), Access: role})
}

func TestConsoleViewerCanReadButNotWrite(t *testing.T) {
	svc, repo, _, _ := newConsoleService()
	ctx := consoleContext(auth.RoleViewer)

	if _, err := svc.ListUsers(ctx, &consolev1.ListUsersRequest{}); err != nil {
		t.Fatalf("viewer ListUsers: %v", err)
	}
	if _, err := svc.UpdateUserStatus(ctx, &consolev1.UpdateUserStatusRequest{Id: 7, Status: 1}); err == nil {
		t.Fatal("viewer UpdateUserStatus should be denied")
	} else if kerrors.Code(err) != http.StatusForbidden {
		t.Fatalf("viewer write code = %d, want 403", kerrors.Code(err))
	}
	if repo.updateCalls != 0 {
		t.Fatalf("repo update calls = %d, want 0", repo.updateCalls)
	}
}

func TestConsoleOperatorWritesAndAudits(t *testing.T) {
	svc, repo, _, audit := newConsoleService()
	ctx := consoleContext(auth.RoleOperator)

	user, err := svc.UpdateUserStatus(ctx, &consolev1.UpdateUserStatusRequest{Id: 7, Status: 1})
	if err != nil {
		t.Fatalf("operator UpdateUserStatus: %v", err)
	}
	if user.Status != 1 {
		t.Fatalf("status = %d, want 1", user.Status)
	}
	if repo.updateCalls != 1 {
		t.Fatalf("repo update calls = %d, want 1", repo.updateCalls)
	}
	if len(audit.entries) != 1 {
		t.Fatalf("audit entries = %d, want 1", len(audit.entries))
	}
	entry := audit.entries[0]
	if entry.Action != "user.status" || entry.Target != "7" || entry.Result != "success" {
		t.Fatalf("unexpected audit entry: %+v", entry)
	}
	if entry.RemoteAddr != "203.0.113.9" {
		t.Fatalf("remote addr = %q, want the client IP", entry.RemoteAddr)
	}
}

func TestConsoleOpsFailureReturns503AndAuditsFailure(t *testing.T) {
	svc, _, ops, audit := newConsoleService()
	ops.reviewErr = biz.ErrOpsUnavailable
	ctx := consoleContext(auth.RoleOperator)

	_, err := svc.ReviewProject(ctx, &consolev1.ReviewProjectRequest{Id: 4001, Decision: "approve"})
	if err == nil {
		t.Fatal("expected the ops error to surface")
	}
	if kerrors.Code(err) != http.StatusServiceUnavailable {
		t.Fatalf("code = %d, want 503 (%v)", kerrors.Code(err), err)
	}
	if len(audit.entries) != 1 || audit.entries[0].Result != "failure" {
		t.Fatalf("audit entries = %+v, want one failure", audit.entries)
	}
}

func TestConsoleAuditFailureDoesNotBlockWrite(t *testing.T) {
	svc, _, _, audit := newConsoleService()
	audit.err = errors.New("audit table missing")
	ctx := consoleContext(auth.RoleOperator)

	if _, err := svc.HandleReport(ctx, &consolev1.HandleReportRequest{
		Id: 9001, Status: "accepted", Note: "ok",
	}); err != nil {
		t.Fatalf("HandleReport should succeed even when audit fails: %v", err)
	}
}

func TestConsolePageSizeIsCapped(t *testing.T) {
	svc, repo, _, _ := newConsoleService()
	ctx := consoleContext(auth.RoleViewer)

	if _, err := svc.ListUsers(ctx, &consolev1.ListUsersRequest{PageSize: 500}); err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if repo.lastUserQuery.Limit != 200 {
		t.Fatalf("limit = %d, want 200", repo.lastUserQuery.Limit)
	}
	if _, err := svc.ListUsers(ctx, &consolev1.ListUsersRequest{}); err != nil {
		t.Fatalf("ListUsers default: %v", err)
	}
	if repo.lastUserQuery.Limit != 20 {
		t.Fatalf("default limit = %d, want 20", repo.lastUserQuery.Limit)
	}
}
