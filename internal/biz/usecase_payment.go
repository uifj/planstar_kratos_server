package biz

import "context"

// PaymentUsecase owns the local mock payment channel. A successful mock pay
// writes the payment.state change and the payment.success outbox row in one
// transaction, so the event can never be lost.
type PaymentUsecase struct {
	repo   PaymentRepo
	orders OrderRepo
}

func NewPaymentUsecase(repo PaymentRepo, orders OrderRepo) *PaymentUsecase {
	return &PaymentUsecase{repo: repo, orders: orders}
}

func (uc *PaymentUsecase) Create(ctx context.Context, userID, orderSn string, amount int64, channel string) (*Payment, error) {
	if orderSn == "" || amount <= 0 {
		return nil, ErrOrderInvalidArgument
	}
	order, err := uc.orders.FindBySn(ctx, orderSn)
	if err != nil {
		return nil, err
	}
	if order.UserID != userID {
		return nil, ErrOrderNotFound
	}
	if order.Status != OrderStatusLocked {
		return nil, ErrOrderStatusConflict
	}
	if order.Total != amount {
		return nil, ErrPaymentAmountMismatch
	}
	if channel == "" {
		channel = "mock"
	}
	return uc.repo.Create(ctx, &Payment{
		ID: newID(), PaymentNo: newPaymentNo(), OrderSn: orderSn,
		UserID: userID, Amount: amount, Channel: channel,
		Status: PaymentStatusPending,
	})
}

func (uc *PaymentUsecase) MockPay(ctx context.Context, userID, paymentNo string) error {
	if paymentNo == "" {
		return ErrOrderInvalidArgument
	}
	p, err := uc.repo.FindByNo(ctx, paymentNo)
	if err != nil {
		return err
	}
	if p.UserID != userID {
		return ErrPaymentNotFound
	}
	if p.Status != PaymentStatusPending {
		return ErrPaymentStatusConflict
	}
	order, err := uc.orders.FindBySn(ctx, p.OrderSn)
	if err != nil {
		return err
	}
	if order.Total != p.Amount {
		return ErrPaymentAmountMismatch
	}
	event := newDomainEvent(EventPaymentSuccess, p.OrderSn, userID)
	event.PaymentNo = p.PaymentNo
	event.Amount = p.Amount
	goods, err := uc.orders.OrderGoods(ctx, order.ID)
	if err != nil {
		return err
	}
	for _, g := range goods {
		event.Items = append(event.Items, EventItem{
			SkuID: g.SkuID, SkuName: g.SkuName, Price: g.SkuPrice, Num: g.Num,
		})
	}
	outbox := &OutboxEvent{
		ID: newID(), EventID: event.ID, EventType: EventPaymentSuccess,
		OrderSn: p.OrderSn, Payload: encodeEvent(event), Status: OutboxPending,
	}
	return uc.repo.MarkSuccessWithEvent(ctx, paymentNo, outbox)
}

// PermissionUsecase owns the RBAC read/write surface.
type PermissionUsecase struct {
	repo PermissionRepo
}

func NewPermissionUsecase(repo PermissionRepo) *PermissionUsecase {
	return &PermissionUsecase{repo: repo}
}

func (uc *PermissionUsecase) List(ctx context.Context) ([]*Permission, error) {
	return uc.repo.List(ctx)
}

func (uc *PermissionUsecase) RoleList(ctx context.Context, role int32) ([]string, error) {
	if role <= 0 {
		return nil, ErrPermissionDenied
	}
	return uc.repo.ListCodesByRole(ctx, role)
}

func (uc *PermissionUsecase) RoleUpdate(ctx context.Context, role int32, codes []string) error {
	if role <= 0 {
		return ErrPermissionDenied
	}
	return uc.repo.ReplaceRole(ctx, role, codes)
}
