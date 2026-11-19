package biz

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// memOrder 是内存测试订单。
type memOrder struct {
	id     int64
	number string
	status int
	amount int64
}

// memPaymentRepo 是内存版支付/退款仓储，行为与 data 层保持一致的幂等/状态规则。
type memPaymentRepo struct {
	orders   map[string]*memOrder
	byID     map[int64]*memOrder
	payments map[string]*PaymentOrder
	refunds  map[int64]*Refund
}

func newMemPaymentRepo() *memPaymentRepo {
	order := &memOrder{id: 1, number: "PN1", status: 2, amount: 1990}
	return &memPaymentRepo{
		orders:   map[string]*memOrder{order.number: order},
		byID:     map[int64]*memOrder{order.id: order},
		payments: map[string]*PaymentOrder{},
		refunds:  map[int64]*Refund{},
	}
}

func payKey(orderNumber, channel string) string { return orderNumber + "|" + channel }

func (m *memPaymentRepo) CreatePayment(_ context.Context, _ int64, in *PayInput, now time.Time) (*PaymentOrder, error) {
	order, ok := m.orders[in.OrderNumber]
	if !ok {
		return nil, ErrNotFound
	}
	if order.status == 6 || order.status == 7 {
		return nil, ErrConflict
	}
	key := payKey(in.OrderNumber, in.Channel)
	if p, ok := m.payments[key]; ok {
		return p, nil
	}
	p := &PaymentOrder{
		PayNo:       "PY" + strings.ToUpper(in.OrderNumber) + in.Channel,
		OrderNumber: in.OrderNumber,
		Amount:      order.amount,
		Channel:     in.Channel,
		Status:      PaymentStatusPending,
		PayParams:   map[string]any{},
		ExpireAt:    now.Add(DefaultPaymentTTL).UnixMilli(),
	}
	m.payments[key] = p
	return p, nil
}

func (m *memPaymentRepo) PaymentDetail(_ context.Context, _ int64, payNo, orderNumber string) (*PaymentOrder, error) {
	for _, p := range m.payments {
		if p.PayNo == payNo || (payNo == "" && p.OrderNumber == orderNumber) {
			return p, nil
		}
	}
	return nil, ErrNotFound
}

func (m *memPaymentRepo) CreateRefund(_ context.Context, _ int64, orderID int64, _ *RefundInput, _ time.Time) (*Refund, error) {
	order, ok := m.byID[orderID]
	if !ok {
		return nil, ErrNotFound
	}
	if r, ok := m.refunds[orderID]; ok {
		return r, nil
	}
	if order.status < 1 || order.status > 5 || order.amount <= 0 {
		return nil, ErrRefundNotAllowed
	}
	r := &Refund{
		RefundNo:            "RF" + order.number,
		OrderID:             orderID,
		RefundableAmount:    order.amount,
		Status:              "applied",
		ExpectedArrivalDays: DefaultRefundArrivalDays,
	}
	m.refunds[orderID] = r
	return r, nil
}

func (m *memPaymentRepo) RefundByOrder(_ context.Context, _ int64, orderID int64) (*Refund, error) {
	if r, ok := m.refunds[orderID]; ok {
		return r, nil
	}
	return nil, ErrNotFound
}

func TestPaymentPayIdempotent(t *testing.T) {
	repo := newMemPaymentRepo()
	uc := NewPaymentUsecase(repo)
	uc.now = func() time.Time { return time.UnixMilli(1_760_000_000_000) }

	in := &PayInput{OrderNumber: "PN1", Channel: "wechat"}
	first, err := uc.Pay(context.Background(), 1, in)
	if err != nil {
		t.Fatalf("first pay err = %v", err)
	}
	second, err := uc.Pay(context.Background(), 1, in)
	if err != nil {
		t.Fatalf("second pay err = %v", err)
	}
	if first.PayNo == "" || first.PayNo != second.PayNo {
		t.Fatalf("payNo first=%q second=%q, want same non-empty", first.PayNo, second.PayNo)
	}
	if second.Status != PaymentStatusPending || second.Amount != 1990 {
		t.Fatalf("payment = %+v, want pending/1990", second)
	}
}

func TestPaymentPayRejectsBadInput(t *testing.T) {
	uc := NewPaymentUsecase(newMemPaymentRepo())
	if _, err := uc.Pay(context.Background(), 1, &PayInput{OrderNumber: "PN1", Channel: "paypal"}); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("bad channel err = %v, want ErrBadRequest", err)
	}
	if _, err := uc.Pay(context.Background(), 1, &PayInput{OrderNumber: "", Channel: "wechat"}); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("empty order err = %v, want ErrBadRequest", err)
	}
}

func TestPaymentDetailExpired(t *testing.T) {
	repo := newMemPaymentRepo()
	uc := NewPaymentUsecase(repo)
	fixed := time.UnixMilli(1_760_000_000_000)
	uc.now = func() time.Time { return fixed }
	repo.payments[payKey("PN1", "wechat")] = &PaymentOrder{
		PayNo: "PY_EXPIRED", OrderNumber: "PN1", Amount: 1990, Channel: "wechat",
		Status: PaymentStatusPending, ExpireAt: fixed.Add(-time.Minute).UnixMilli(),
	}

	p, err := uc.PayDetail(context.Background(), 1, "PY_EXPIRED", "")
	if err != nil {
		t.Fatalf("payDetail err = %v", err)
	}
	if p.Status != PaymentStatusExpired {
		t.Fatalf("status = %q, want expired", p.Status)
	}
}

func TestRefundIdempotentAndNotAllowed(t *testing.T) {
	repo := newMemPaymentRepo()
	uc := NewPaymentUsecase(repo)
	in := &RefundInput{ReasonCode: "notNeeded", ReasonText: "买错了"}

	first, err := uc.Refund(context.Background(), 1, 1, in)
	if err != nil {
		t.Fatalf("first refund err = %v", err)
	}
	second, err := uc.Refund(context.Background(), 1, 1, in)
	if err != nil {
		t.Fatalf("second refund err = %v", err)
	}
	if first.RefundNo == "" || first.RefundNo != second.RefundNo {
		t.Fatalf("refundNo first=%q second=%q, want same non-empty", first.RefundNo, second.RefundNo)
	}
	if first.RefundableAmount != 1990 || first.ExpectedArrivalDays != DefaultRefundArrivalDays {
		t.Fatalf("refund = %+v, want amount 1990 / 3 days", first)
	}

	// 订单已关闭（且尚无退款单）：不可退款。
	closed := &memOrder{id: 2, number: "PN2", status: 6, amount: 990}
	repo.orders[closed.number] = closed
	repo.byID[closed.id] = closed
	if _, err := uc.Refund(context.Background(), 1, closed.id, in); !errors.Is(err, ErrRefundNotAllowed) {
		t.Fatalf("closed order refund err = %v, want ErrRefundNotAllowed", err)
	}
}

func TestRefundRejectsBadReason(t *testing.T) {
	uc := NewPaymentUsecase(newMemPaymentRepo())
	if _, err := uc.Refund(context.Background(), 1, 1, &RefundInput{ReasonCode: "unknown"}); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("bad reason err = %v, want ErrBadRequest", err)
	}
}
