package biz

import (
	"context"
	"log/slog"
)

// OrderInput 下单入参（金额单位为分，落库时转元）。
type OrderInput struct {
	ActivityID      int64
	ActivityEventID int64
	OrderType       int
	PayAmount       int64
	TotalAmount     int64
	Name            string
	Phone           string
	AddressID       int64
}

// OrderRepo 是订单仓储端口。
type OrderRepo interface {
	Orders(ctx context.Context, userID int64, status, orderType, pageIndex, pageSize int) ([]*TicketOrder, error)
	OrderByNumber(ctx context.Context, userID int64, orderNumber string) (*TicketOrder, error)
	CreateOrder(ctx context.Context, userID int64, in *OrderInput) (*TicketOrder, error)
}

// OrderUsecase 订单用例。
type OrderUsecase struct {
	repo   OrderRepo
	events EventPublisher
	log    *slog.Logger
}

func NewOrderUsecase(repo OrderRepo, events EventPublisher, log *slog.Logger) *OrderUsecase {
	if events == nil {
		events = NopEventPublisher{}
	}
	if log == nil {
		log = slog.Default()
	}
	return &OrderUsecase{repo: repo, events: events, log: log}
}

func (uc *OrderUsecase) Orders(ctx context.Context, userID int64, status, orderType, pageIndex, pageSize int) ([]*TicketOrder, error) {
	if pageIndex <= 0 {
		pageIndex = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	items, err := uc.repo.Orders(ctx, userID, status, orderType, pageIndex, pageSize)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*TicketOrder{}
	}
	return items, nil
}

func (uc *OrderUsecase) OrderByNumber(ctx context.Context, userID int64, orderNumber string) (*TicketOrder, error) {
	if orderNumber == "" {
		return nil, ErrBadRequest
	}
	return uc.repo.OrderByNumber(ctx, userID, orderNumber)
}

func (uc *OrderUsecase) MakeOrder(ctx context.Context, userID int64, in *OrderInput) (*TicketOrder, error) {
	if in.ActivityID <= 0 {
		return nil, ErrBadRequest
	}
	if in.OrderType == 0 {
		in.OrderType = 1
	}
	order, err := uc.repo.CreateOrder(ctx, userID, in)
	if err != nil {
		return nil, err
	}
	// 异步事件：订单创建通知（失败不影响主流程，由 mq 适配器记录重试）。
	if err := uc.events.Publish(ctx, NewEvent(EventOrderCreated, userID, map[string]any{
		"orderId":     order.OrderID,
		"orderNumber": order.OrderNumber,
		"activityId":  order.ShowID,
		"payAmount":   order.PayAmount,
	})); err != nil {
		uc.log.WarnContext(ctx, "publish order.created failed", "orderNumber", order.OrderNumber, "err", err)
	}
	return order, nil
}
