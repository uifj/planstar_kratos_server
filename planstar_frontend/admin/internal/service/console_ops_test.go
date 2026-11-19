package service

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	kerrors "github.com/go-kratos/kratos/v3/errors"

	consolev1 "planstar/admin/api/planstar/admin/v1"
	"planstar/admin/internal/biz"
	"planstar/admin/pkg/auth"
)

// statusSetter adapts each status write RPC to a uniform signature so the gate,
// validation, delegation and audit assertions can be table-driven across them.
type statusSetter struct {
	name   string
	action string
	set    func(*ConsoleService, context.Context, int64, int32) error
}

func statusSetters() []statusSetter {
	return []statusSetter{
		{"feed", "feed.status", func(s *ConsoleService, c context.Context, id int64, st int32) error {
			_, e := s.SetFeedStatus(c, &consolev1.SetFeedStatusRequest{Id: id, Status: st})
			return e
		}},
		{"goods", "goods.status", func(s *ConsoleService, c context.Context, id int64, st int32) error {
			_, e := s.SetGoodsStatus(c, &consolev1.SetGoodsStatusRequest{Id: id, Status: st})
			return e
		}},
		{"sku", "goods_sku.status", func(s *ConsoleService, c context.Context, id int64, st int32) error {
			_, e := s.SetGoodsSkuStatus(c, &consolev1.SetGoodsSkuStatusRequest{Id: id, Status: st})
			return e
		}},
	}
}

func reviewRefund(s *ConsoleService, c context.Context, decision string) error {
	_, e := s.ReviewRefund(c, &consolev1.ReviewRefundRequest{Id: 2, Decision: decision})
	return e
}

func TestConsoleSetStatusDelegatesAndAudits(t *testing.T) {
	for _, setter := range statusSetters() {
		for _, status := range []int32{0, 1} {
			t.Run(fmt.Sprintf("%s/status%d", setter.name, status), func(t *testing.T) {
				svc, _, ops, audit := newConsoleService()
				ctx := consoleContext(auth.RoleOperator)
				if err := setter.set(svc, ctx, 30, status); err != nil {
					t.Fatalf("unexpected error %v", err)
				}
				if ops.lastSetID != 30 || ops.lastSetVal != status {
					t.Fatalf("ops delegation = %d/%d, want 30/%d", ops.lastSetID, ops.lastSetVal, status)
				}
				if len(audit.entries) != 1 {
					t.Fatalf("audit entries = %d, want 1", len(audit.entries))
				}
				e := audit.entries[0]
				if e.Action != setter.action || e.Target != "30" || e.Result != "success" {
					t.Fatalf("audit = %+v, want %s/30/success", e, setter.action)
				}
			})
		}
	}
}

func TestConsoleSetStatusOutOfRangeAuditsFailure(t *testing.T) {
	for _, setter := range statusSetters() {
		for _, status := range []int32{-1, 2} {
			t.Run(fmt.Sprintf("%s/reject%d", setter.name, status), func(t *testing.T) {
				svc, _, _, audit := newConsoleService()
				ctx := consoleContext(auth.RoleOperator)
				err := setter.set(svc, ctx, 1, status)
				if kerrors.Code(err) != http.StatusBadRequest {
					t.Fatalf("code = %d, want 400", kerrors.Code(err))
				}
				if len(audit.entries) != 1 || audit.entries[0].Result != "failure" {
					t.Fatalf("audit = %+v, want 1 failure", audit.entries)
				}
			})
		}
	}
}

func TestConsoleOpsWritesRejectViewer(t *testing.T) {
	calls := []struct {
		name string
		call func(*ConsoleService, context.Context) error
	}{
		{"feed", func(s *ConsoleService, c context.Context) error {
			_, e := s.SetFeedStatus(c, &consolev1.SetFeedStatusRequest{Id: 30, Status: 0})
			return e
		}},
		{"goods", func(s *ConsoleService, c context.Context) error {
			_, e := s.SetGoodsStatus(c, &consolev1.SetGoodsStatusRequest{Id: 1, Status: 1})
			return e
		}},
		{"sku", func(s *ConsoleService, c context.Context) error {
			_, e := s.SetGoodsSkuStatus(c, &consolev1.SetGoodsSkuStatusRequest{Id: 2, Status: 1})
			return e
		}},
		{"refund", func(s *ConsoleService, c context.Context) error {
			return reviewRefund(s, c, "approve")
		}},
	}
	for _, tc := range calls {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, ops, audit := newConsoleService()
			ctx := consoleContext(auth.RoleViewer)
			if err := tc.call(svc, ctx); kerrors.Code(err) != http.StatusForbidden {
				t.Fatalf("code = %d, want 403", kerrors.Code(err))
			}
			if ops.lastSetID != 0 || ops.lastDecision != "" {
				t.Fatal("ops must not be called for viewer")
			}
			if len(audit.entries) != 0 {
				t.Fatalf("audit entries = %d, want 0 (gate rejects before audit)", len(audit.entries))
			}
		})
	}
}

func TestConsoleReviewRefundDecision(t *testing.T) {
	cases := []struct {
		decision string
		wantCode int
	}{
		{"approve", http.StatusOK},
		{"reject", http.StatusOK},
		{"maybe", http.StatusBadRequest},
		{"", http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run("decision_"+tc.decision, func(t *testing.T) {
			svc, _, ops, audit := newConsoleService()
			ctx := consoleContext(auth.RoleOperator)
			err := reviewRefund(svc, ctx, tc.decision)
			if kerrors.Code(err) != tc.wantCode {
				t.Fatalf("code = %d, want %d", kerrors.Code(err), tc.wantCode)
			}
			wantResult := "success"
			if tc.wantCode != http.StatusOK {
				wantResult = "failure"
			}
			if len(audit.entries) != 1 || audit.entries[0].Action != "refund.review" || audit.entries[0].Result != wantResult {
				t.Fatalf("audit = %+v, want refund.review/%s", audit.entries, wantResult)
			}
			if tc.wantCode == http.StatusOK && ops.lastDecision != tc.decision {
				t.Fatalf("decision = %q, want %q", ops.lastDecision, tc.decision)
			}
		})
	}
}

func TestConsoleReviewRefundSuccessReturnsOrderStatus(t *testing.T) {
	svc, _, _, _ := newConsoleService()
	ctx := consoleContext(auth.RoleOperator)
	review, err := svc.ReviewRefund(ctx, &consolev1.ReviewRefundRequest{Id: 2, Decision: "approve"})
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if review.Status != "success" || review.OrderStatus != 7 {
		t.Fatalf("review = %+v, want success + order status 7", review)
	}
}

func TestConsoleOpsWriteFailureReturns503(t *testing.T) {
	for _, setter := range statusSetters() {
		t.Run(setter.name, func(t *testing.T) {
			svc, _, ops, audit := newConsoleService()
			ops.statusErr = biz.ErrOpsUnavailable
			ctx := consoleContext(auth.RoleOperator)
			if err := setter.set(svc, ctx, 30, 1); kerrors.Code(err) != http.StatusServiceUnavailable {
				t.Fatalf("code = %d, want 503", kerrors.Code(err))
			}
			if len(audit.entries) != 1 || audit.entries[0].Result != "failure" {
				t.Fatalf("audit = %+v, want 1 failure", audit.entries)
			}
		})
	}
	t.Run("refund", func(t *testing.T) {
		svc, _, ops, audit := newConsoleService()
		ops.refundErr = biz.ErrOpsUnavailable
		ctx := consoleContext(auth.RoleOperator)
		if err := reviewRefund(svc, ctx, "approve"); kerrors.Code(err) != http.StatusServiceUnavailable {
			t.Fatalf("code = %d, want 503", kerrors.Code(err))
		}
		if len(audit.entries) != 1 || audit.entries[0].Result != "failure" {
			t.Fatalf("audit = %+v, want 1 failure", audit.entries)
		}
	})
}
