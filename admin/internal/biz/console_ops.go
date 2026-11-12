package biz

import (
	"context"
	"strings"
)

// RefundReview is the result of reviewing a refund (RefundAmount in cents).
type RefundReview struct {
	RefundID     int64
	RefundNo     string
	OrderID      int64
	Status       string
	OrderStatus  int32
	RefundAmount int64
}

// SetFeedStatus hides (0) or shows (1) a community feed through the ops service.
func (uc *ConsoleUsecase) SetFeedStatus(ctx context.Context, id int64, status int32) error {
	if id <= 0 {
		return ErrFeedNotFound
	}
	if status < 0 || status > 1 {
		return ErrInvalidStatus
	}
	return uc.ops.SetFeedStatus(ctx, id, status)
}

// SetGoodsStatus puts a shop goods off (0) or on (1) shelf through the ops service.
func (uc *ConsoleUsecase) SetGoodsStatus(ctx context.Context, id int64, status int32) error {
	if id <= 0 {
		return ErrGoodsNotFound
	}
	if status < 0 || status > 1 {
		return ErrInvalidStatus
	}
	return uc.ops.SetGoodsStatus(ctx, id, status)
}

// SetGoodsSkuStatus puts a goods SKU off (0) or on (1) shelf through the ops service.
func (uc *ConsoleUsecase) SetGoodsSkuStatus(ctx context.Context, id int64, status int32) error {
	if id <= 0 {
		return ErrSkuNotFound
	}
	if status < 0 || status > 1 {
		return ErrInvalidStatus
	}
	return uc.ops.SetGoodsSkuStatus(ctx, id, status)
}

// ReviewRefund approves or rejects a refund through the ops service.
func (uc *ConsoleUsecase) ReviewRefund(ctx context.Context, id int64, decision string) (*RefundReview, error) {
	if id <= 0 {
		return nil, ErrRefundNotFound
	}
	decision = strings.TrimSpace(decision)
	if decision != "approve" && decision != "reject" {
		return nil, ErrInvalidDecision
	}
	return uc.ops.ReviewRefund(ctx, id, decision)
}