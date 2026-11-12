package data

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	kerrors "github.com/go-kratos/kratos/v3/errors"

	"planstar/admin/internal/biz"
	"planstar/admin/internal/conf"
)

// newOpsClientForTest points an opsClient at a stub ops server so the transport
// mapping (2xx / 400 / 404 / 5xx / unreachable) is exercised without the real
// business service.
func newOpsClientForTest(t *testing.T, h http.HandlerFunc) biz.OpsClient {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return NewOpsClient(&conf.Planstar{OpsBaseUrl: srv.URL, OpsToken: "test-token"})
}

func TestOpsClientMapsSuccess(t *testing.T) {
	c := newOpsClientForTest(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q, want Bearer test-token", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"refundId":7,"refundNo":"R7","orderId":8,"status":"success","orderStatus":7,"refundAmount":5000}`))
	})
	out, err := c.ReviewRefund(context.Background(), 7, "approve")
	if err != nil {
		t.Fatalf("ReviewRefund: %v", err)
	}
	if out.RefundID != 7 || out.OrderID != 8 || out.Status != "success" || out.OrderStatus != 7 || out.RefundAmount != 5000 {
		t.Fatalf("review = %+v, want decoded ops reply", out)
	}
}

func TestOpsClientMapsBadRequestPreservingMessageAndReasons(t *testing.T) {
	c := newOpsClientForTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"REFUND_NOT_ALLOWED","message":"退款单当前状态不可审核","reasons":[{"rule":"blacklist","keyword":"foo"}]}`))
	})
	err := c.SetFeedStatus(context.Background(), 1, 0)
	if kerrors.Code(err) != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400", kerrors.Code(err))
	}
	msg := kerrors.FromError(err).Message
	if !strings.Contains(msg, "退款单当前状态不可审核") || !strings.Contains(msg, "blacklist(foo)") {
		t.Fatalf("message = %q, want ops message + rule(keyword) appended", msg)
	}
}

func TestOpsClientMapsNotFound(t *testing.T) {
	c := newOpsClientForTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	cases := []struct {
		name string
		call func() error
		want error
	}{
		{"feed", func() error { return c.SetFeedStatus(context.Background(), 1, 0) }, biz.ErrFeedNotFound},
		{"goods", func() error { return c.SetGoodsStatus(context.Background(), 1, 0) }, biz.ErrGoodsNotFound},
		{"sku", func() error { return c.SetGoodsSkuStatus(context.Background(), 1, 0) }, biz.ErrSkuNotFound},
		{"refund", func() error { _, e := c.ReviewRefund(context.Background(), 1, "approve"); return e }, biz.ErrRefundNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.call(); !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestOpsClientMapsUnavailable(t *testing.T) {
	t.Run("server 500", func(t *testing.T) {
		c := newOpsClientForTest(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})
		if err := c.SetFeedStatus(context.Background(), 1, 0); !errors.Is(err, biz.ErrOpsUnavailable) {
			t.Fatalf("err = %v, want ErrOpsUnavailable", err)
		}
	})
	t.Run("empty base url", func(t *testing.T) {
		c := NewOpsClient(&conf.Planstar{})
		if err := c.SetFeedStatus(context.Background(), 1, 0); !errors.Is(err, biz.ErrOpsUnavailable) {
			t.Fatalf("err = %v, want ErrOpsUnavailable", err)
		}
	})
	t.Run("unreachable server", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		url := srv.URL
		srv.Close() // force a transport error
		c := NewOpsClient(&conf.Planstar{OpsBaseUrl: url, OpsToken: "t"})
		if err := c.SetFeedStatus(context.Background(), 1, 0); !errors.Is(err, biz.ErrOpsUnavailable) {
			t.Fatalf("err = %v, want ErrOpsUnavailable", err)
		}
	})
}
