package biz

import (
	"context"
	"log/slog"
)

// OrderInput 下单入参（金额单位为分；PayAmount/TotalAmount 仅保留兼容，落库以服务端报价为准）。
type OrderInput struct {
	ActivityID       int64
	ActivityEventID  int64
	TicketCategoryID int64
	OrderType        int
	Quantity         int
	// Amount 是客户端提交的应付金额（分），仅用于与服务端重算结果比对。
	Amount      int64
	PayAmount   int64
	TotalAmount int64
	// QuoteID / QuoteVersion 指向 confirm 接口下发的报价快照。
	QuoteID      string
	QuoteVersion int
	// IdempotencyKey 来自请求头 Idempotency-Key，同 key 重复请求返回同一订单。
	IdempotencyKey string
	Name           string
	Phone          string
	AddressID      int64
}

// OrderRepo 是订单仓储端口（同时提供报价快照存储与演出权威数据读取）。
type OrderRepo interface {
	Orders(ctx context.Context, userID int64, status, orderType, pageIndex, pageSize int) ([]*TicketOrder, error)
	OrderByNumber(ctx context.Context, userID int64, orderNumber string) (*TicketOrder, error)
	// CreateOrder 在单事务内完成幂等校验、库存扣减与订单写入；quote 为服务端权威报价。
	CreateOrder(ctx context.Context, userID int64, in *OrderInput, quote *QuoteSnapshot) (*TicketOrder, error)
	QuoteStore
	CatalogReader
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

// MakeOrder 服务端权威定价下单：解析报价快照 → 重算金额 → 事务内扣库存并落单。
func (uc *OrderUsecase) MakeOrder(ctx context.Context, userID int64, in *OrderInput) (*TicketOrder, error) {
	if in == nil {
		return nil, ErrBadRequest
	}
	if in.ActivityID <= 0 && in.QuoteID == "" {
		return nil, ErrBadRequest
	}
	if in.Quantity > MaxQuantityPerOrder {
		return nil, ErrLimitExceeded
	}
	if in.OrderType == 0 {
		in.OrderType = 1
	}
	quote, err := uc.resolveQuote(ctx, userID, in)
	if err != nil {
		return nil, err
	}
	// 以服务端报价覆盖客户端金额与演出维度，禁止信任客户端金额。
	in.ActivityID = quote.ActivityID
	if quote.ActivityEventID > 0 {
		in.ActivityEventID = quote.ActivityEventID
	}
	in.TicketCategoryID = quote.TicketCategoryID
	in.Quantity = quote.Quantity
	in.PayAmount = quote.TotalAmount
	in.TotalAmount = quote.TotalAmount

	order, err := uc.repo.CreateOrder(ctx, userID, in, quote)
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
