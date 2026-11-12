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
	users             map[int64]*biz.PlanstarUser
	lastUserQuery     biz.UserQuery
	updateCalls       int
	lastIdeaQuery     biz.IdeaQuery
	lastGoodsQuery    biz.GoodsQuery
	lastGoodsOrdQuery biz.GoodsOrderQuery
	lastPayOrdQuery   biz.PayOrderQuery
	lastRefundQuery   biz.RefundQuery
	lastFeedQuery     biz.FeedQuery
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

func (r *fakeConsoleRepo) Dashboard(context.Context) (*biz.Dashboard, error) {
	return &biz.Dashboard{
		UsersTotal: 1,
		Trends: []biz.TrendSeries{
			{Name: "orders", Points: []biz.DailyCount{{Date: "2026-11-12", Count: 2}}},
		},
	}, nil
}

func (r *fakeConsoleRepo) ListIdeas(_ context.Context, q biz.IdeaQuery) ([]*biz.Idea, error) {
	r.lastIdeaQuery = q
	return nil, nil
}

func (r *fakeConsoleRepo) GetIdea(context.Context, int64) (*biz.IdeaDetail, error) {
	return nil, biz.ErrIdeaNotFound
}

func (r *fakeConsoleRepo) ListGoods(_ context.Context, q biz.GoodsQuery) ([]*biz.Goods, error) {
	r.lastGoodsQuery = q
	return nil, nil
}

func (r *fakeConsoleRepo) GetGoods(context.Context, int64) (*biz.GoodsDetail, error) {
	return nil, biz.ErrGoodsNotFound
}

func (r *fakeConsoleRepo) ListGoodsOrders(_ context.Context, q biz.GoodsOrderQuery) ([]*biz.GoodsOrder, error) {
	r.lastGoodsOrdQuery = q
	return nil, nil
}

func (r *fakeConsoleRepo) GetOrder(context.Context, int64) (*biz.OrderDetail, error) {
	return nil, biz.ErrOrderNotFound
}

func (r *fakeConsoleRepo) ListPayOrders(_ context.Context, q biz.PayOrderQuery) ([]*biz.PayOrder, error) {
	r.lastPayOrdQuery = q
	return nil, nil
}

func (r *fakeConsoleRepo) ListRefunds(_ context.Context, q biz.RefundQuery) ([]*biz.Refund, error) {
	r.lastRefundQuery = q
	return nil, nil
}

func (r *fakeConsoleRepo) ListFeeds(_ context.Context, q biz.FeedQuery) ([]*biz.Feed, error) {
	r.lastFeedQuery = q
	return nil, nil
}

func (r *fakeConsoleRepo) GetFeed(context.Context, int64) (*biz.FeedDetail, error) {
	return nil, biz.ErrFeedNotFound
}

type fakeOps struct {
	reviewErr    error
	reportErr    error
	reviewResult *biz.ReviewResult
	reportResult *biz.Report
	lastDecision string
	lastStatus   string
	statusErr    error
	lastSetID    int64
	lastSetVal   int32
	refundErr    error
	refundResult *biz.RefundReview
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

func (f *fakeOps) SetFeedStatus(_ context.Context, id int64, status int32) error {
	f.lastSetID, f.lastSetVal = id, status
	return f.statusErr
}

func (f *fakeOps) SetGoodsStatus(_ context.Context, id int64, status int32) error {
	f.lastSetID, f.lastSetVal = id, status
	return f.statusErr
}

func (f *fakeOps) SetGoodsSkuStatus(_ context.Context, id int64, status int32) error {
	f.lastSetID, f.lastSetVal = id, status
	return f.statusErr
}

func (f *fakeOps) ReviewRefund(_ context.Context, id int64, decision string) (*biz.RefundReview, error) {
	f.lastDecision = decision
	if f.refundErr != nil {
		return nil, f.refundErr
	}
	if f.refundResult != nil {
		return f.refundResult, nil
	}
	return &biz.RefundReview{RefundID: id, Status: "success", OrderStatus: 7}, nil
}

type fakeAuditRepo struct {
	entries   []*biz.AuditEntry
	err       error
	lastQuery biz.AuditQuery
}

func (r *fakeAuditRepo) Record(_ context.Context, entry *biz.AuditEntry) error {
	copied := *entry
	r.entries = append(r.entries, &copied)
	return r.err
}

func (r *fakeAuditRepo) ListAuditLogs(_ context.Context, q biz.AuditQuery) ([]*biz.AuditLog, error) {
	r.lastQuery = q
	return []*biz.AuditLog{{ID: 1, Actor: "admin", Action: "user.status", Target: "7", Result: "success"}}, nil
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

func TestConsoleUserStatusRejectsOutOfRange(t *testing.T) {
	svc, repo, _, audit := newConsoleService()
	ctx := consoleContext(auth.RoleOperator)

	for _, status := range []int32{-1, 2, 9} {
		if _, err := svc.UpdateUserStatus(ctx, &consolev1.UpdateUserStatusRequest{Id: 7, Status: status}); err == nil {
			t.Fatalf("status %d should be rejected", status)
		} else if kerrors.Code(err) != http.StatusBadRequest {
			t.Fatalf("status %d code = %d, want 400", status, kerrors.Code(err))
		}
	}
	if repo.updateCalls != 0 {
		t.Fatalf("repo update calls = %d, want 0", repo.updateCalls)
	}
	if len(audit.entries) != 3 {
		t.Fatalf("audit entries = %d, want 3 failures", len(audit.entries))
	}
	for _, entry := range audit.entries {
		if entry.Result != "failure" {
			t.Fatalf("audit entry result = %q, want failure", entry.Result)
		}
	}
	// 0（禁用）是合法值，不能被上界校验误伤。
	if _, err := svc.UpdateUserStatus(ctx, &consolev1.UpdateUserStatusRequest{Id: 7, Status: 0}); err != nil {
		t.Fatalf("status 0 should be allowed: %v", err)
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

func TestConsoleDashboardRequiresViewer(t *testing.T) {
	svc, _, _, _ := newConsoleService()

	dashboard, err := svc.GetDashboard(consoleContext(auth.RoleViewer), &consolev1.GetDashboardRequest{})
	if err != nil {
		t.Fatalf("viewer GetDashboard: %v", err)
	}
	if dashboard.UsersTotal != 1 || len(dashboard.Trends) != 1 || len(dashboard.Trends[0].Points) != 1 {
		t.Fatalf("unexpected dashboard: %+v", dashboard)
	}
	if _, err := svc.GetDashboard(context.Background(), &consolev1.GetDashboardRequest{}); err == nil {
		t.Fatal("unauthenticated GetDashboard should be denied")
	}
}

func TestConsoleAuditLogsAdminOnly(t *testing.T) {
	svc, _, _, audit := newConsoleService()

	if _, err := svc.ListAuditLogs(consoleContext(auth.RoleOperator), &consolev1.ListAuditLogsRequest{}); err == nil {
		t.Fatal("operator ListAuditLogs should be denied")
	} else if kerrors.Code(err) != http.StatusForbidden {
		t.Fatalf("operator audit code = %d, want 403", kerrors.Code(err))
	}
	set, err := svc.ListAuditLogs(consoleContext(auth.RoleAdmin), &consolev1.ListAuditLogsRequest{Result: "success"})
	if err != nil {
		t.Fatalf("admin ListAuditLogs: %v", err)
	}
	if len(set.Logs) != 1 || set.Logs[0].Action != "user.status" {
		t.Fatalf("unexpected audit set: %+v", set)
	}
	if audit.lastQuery.Result != "success" {
		t.Fatalf("result filter = %q, want success", audit.lastQuery.Result)
	}
	if _, err := svc.ListAuditLogs(consoleContext(auth.RoleAdmin), &consolev1.ListAuditLogsRequest{Result: "bogus"}); err == nil {
		t.Fatal("invalid result filter should be rejected")
	}
}

func int32Ptr(v int32) *int32 { return &v }

// Every expanded read must require authentication but only viewer access: an
// unauthenticated call is 401, a viewer call reaches the repo (list succeeds,
// get returns the repo's 404 rather than a 403).
func TestConsoleExpandReadsRequireViewer(t *testing.T) {
	svc, _, _, _ := newConsoleService()
	viewer := consoleContext(auth.RoleViewer)

	calls := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"ListIdeas", func(c context.Context) error { _, e := svc.ListIdeas(c, &consolev1.ListIdeasRequest{}); return e }},
		{"GetIdea", func(c context.Context) error { _, e := svc.GetIdea(c, &consolev1.GetIdeaRequest{Id: 1}); return e }},
		{"ListGoods", func(c context.Context) error { _, e := svc.ListGoods(c, &consolev1.ListGoodsRequest{}); return e }},
		{"GetGoods", func(c context.Context) error { _, e := svc.GetGoods(c, &consolev1.GetGoodsRequest{Id: 1}); return e }},
		{"ListGoodsOrders", func(c context.Context) error { _, e := svc.ListGoodsOrders(c, &consolev1.ListGoodsOrdersRequest{}); return e }},
		{"GetOrder", func(c context.Context) error { _, e := svc.GetOrder(c, &consolev1.GetOrderRequest{Id: 1}); return e }},
		{"ListPayOrders", func(c context.Context) error { _, e := svc.ListPayOrders(c, &consolev1.ListPayOrdersRequest{}); return e }},
		{"ListRefunds", func(c context.Context) error { _, e := svc.ListRefunds(c, &consolev1.ListRefundsRequest{}); return e }},
		{"ListFeeds", func(c context.Context) error { _, e := svc.ListFeeds(c, &consolev1.ListFeedsRequest{}); return e }},
		{"GetFeed", func(c context.Context) error { _, e := svc.GetFeed(c, &consolev1.GetFeedRequest{Id: 1}); return e }},
	}
	for _, call := range calls {
		if err := call.fn(context.Background()); err == nil {
			t.Fatalf("%s unauthenticated should be denied", call.name)
		} else if kerrors.Code(err) != http.StatusUnauthorized {
			t.Fatalf("%s unauthenticated code = %d, want 401", call.name, kerrors.Code(err))
		}
		err := call.fn(viewer)
		if call.name == "GetIdea" || call.name == "GetGoods" || call.name == "GetOrder" || call.name == "GetFeed" {
			if kerrors.Code(err) != http.StatusNotFound {
				t.Fatalf("%s viewer code = %d, want 404 from repo (gate passed)", call.name, kerrors.Code(err))
			}
		} else if err != nil {
			t.Fatalf("%s viewer: %v", call.name, err)
		}
	}
}

func TestConsoleExpandRejectsBadFilters(t *testing.T) {
	svc, repo, _, _ := newConsoleService()
	ctx := consoleContext(auth.RoleViewer)

	bad := []struct {
		name string
		fn   func() error
	}{
		{"ideas publish_status", func() error { _, e := svc.ListIdeas(ctx, &consolev1.ListIdeasRequest{PublishStatus: int32Ptr(4)}); return e }},
		{"goods status", func() error { _, e := svc.ListGoods(ctx, &consolev1.ListGoodsRequest{Status: int32Ptr(2)}); return e }},
		{"feeds status", func() error { _, e := svc.ListFeeds(ctx, &consolev1.ListFeedsRequest{Status: int32Ptr(2)}); return e }},
		{"goods-orders status", func() error { _, e := svc.ListGoodsOrders(ctx, &consolev1.ListGoodsOrdersRequest{Status: "bogus"}); return e }},
		{"pay-orders status", func() error { _, e := svc.ListPayOrders(ctx, &consolev1.ListPayOrdersRequest{Status: "bogus"}); return e }},
		{"refunds status", func() error { _, e := svc.ListRefunds(ctx, &consolev1.ListRefundsRequest{Status: "bogus"}); return e }},
	}
	for _, call := range bad {
		if err := call.fn(); err == nil {
			t.Fatalf("%s invalid filter should be rejected", call.name)
		} else if kerrors.Code(err) != http.StatusBadRequest {
			t.Fatalf("%s code = %d, want 400", call.name, kerrors.Code(err))
		}
	}

	// Boundary values must survive validation and reach the repo.
	if _, err := svc.ListIdeas(ctx, &consolev1.ListIdeasRequest{PublishStatus: int32Ptr(3)}); err != nil {
		t.Fatalf("ideas publish_status 3: %v", err)
	}
	if repo.lastIdeaQuery.Status == nil || *repo.lastIdeaQuery.Status != 3 {
		t.Fatalf("idea status filter = %v, want 3", repo.lastIdeaQuery.Status)
	}
	if _, err := svc.ListGoods(ctx, &consolev1.ListGoodsRequest{Status: int32Ptr(0)}); err != nil {
		t.Fatalf("goods status 0: %v", err)
	}
	if repo.lastGoodsQuery.Status == nil || *repo.lastGoodsQuery.Status != 0 {
		t.Fatalf("goods status filter = %v, want 0", repo.lastGoodsQuery.Status)
	}
	if _, err := svc.ListFeeds(ctx, &consolev1.ListFeedsRequest{Status: int32Ptr(0)}); err != nil {
		t.Fatalf("feeds status 0: %v", err)
	}
	if repo.lastFeedQuery.Status == nil || *repo.lastFeedQuery.Status != 0 {
		t.Fatalf("feed status filter = %v, want 0", repo.lastFeedQuery.Status)
	}
	if _, err := svc.ListRefunds(ctx, &consolev1.ListRefundsRequest{Status: "applied"}); err != nil {
		t.Fatalf("refunds applied: %v", err)
	}
	if repo.lastRefundQuery.Status != "applied" {
		t.Fatalf("refund status filter = %q, want applied", repo.lastRefundQuery.Status)
	}
}
