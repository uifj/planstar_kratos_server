// Package-local inbound event adapters.
//
// internal/mq decodes an AMQP body into *biz.DomainEvent and calls one of the
// pure-Go handler ports declared in internal/biz/event.go. These adapters
// implement those ports by delegating to usecases, so mq stays decoupled from
// concrete usecases and the DTO -> DO -> PO arrows are never reversed.
package service

import (
	"context"

	"github.com/go-kratos/kratos-layout/internal/biz"
)

type orderEvents struct{ orders *biz.OrderUsecase }

func NewOrderEvents(orders *biz.OrderUsecase) biz.OrderEventHandler {
	return &orderEvents{orders: orders}
}

func (h *orderEvents) HandleInventoryLocked(ctx context.Context, e *biz.DomainEvent) error {
	return h.orders.HandleInventoryLocked(ctx, e)
}

func (h *orderEvents) HandleInventoryFailed(ctx context.Context, e *biz.DomainEvent) error {
	return h.orders.HandleInventoryFailed(ctx, e)
}

func (h *orderEvents) HandlePaymentSuccess(ctx context.Context, e *biz.DomainEvent) error {
	return h.orders.HandlePaymentSuccess(ctx, e)
}

type timeoutHandler struct{ orders *biz.OrderUsecase }

func NewTimeoutHandler(orders *biz.OrderUsecase) biz.OrderTimeoutHandler {
	return &timeoutHandler{orders: orders}
}

func (h *timeoutHandler) CancelTimedOut(ctx context.Context, before int64, limit int) (int, error) {
	return h.orders.CancelTimedOut(ctx, before, limit)
}

type inventoryEvents struct{ inventory *biz.InventoryUsecase }

func NewInventoryEvents(inv *biz.InventoryUsecase) biz.InventoryEventHandler {
	return &inventoryEvents{inventory: inv}
}

func (h *inventoryEvents) HandleOrderCreated(ctx context.Context, e *biz.DomainEvent) error {
	return h.inventory.HandleOrderCreated(ctx, e)
}

func (h *inventoryEvents) HandleOrderPaid(ctx context.Context, e *biz.DomainEvent) error {
	return h.inventory.HandleOrderPaid(ctx, e)
}

type goodsEvents struct{ goods *biz.GoodsUsecase }

func NewGoodsEvents(g *biz.GoodsUsecase) biz.GoodsEventHandler {
	return &goodsEvents{goods: g}
}

func (h *goodsEvents) HandleOrderPaid(ctx context.Context, e *biz.DomainEvent) error {
	return h.goods.HandleOrderPaid(ctx, e)
}
