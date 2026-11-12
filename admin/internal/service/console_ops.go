package service

import (
	"context"
	"strconv"

	consolev1 "planstar/admin/api/planstar/admin/v1"
)

// SetFeedStatus hides or shows a community feed through the ops service.
func (s *ConsoleService) SetFeedStatus(ctx context.Context, req *consolev1.SetFeedStatusRequest) (*consolev1.StatusReply, error) {
	a, err := s.operator(ctx)
	if err != nil {
		return nil, err
	}
	target := strconv.FormatInt(req.Id, 10)
	if err := s.uc.SetFeedStatus(ctx, req.Id, req.Status); err != nil {
		s.auditWrite(ctx, a, "feed.status", target, "failure", err.Error())
		return nil, err
	}
	s.auditWrite(ctx, a, "feed.status", target, "success", strconv.FormatInt(int64(req.Status), 10))
	return &consolev1.StatusReply{Id: req.Id, Status: req.Status}, nil
}

// SetGoodsStatus puts a shop goods on or off shelf through the ops service.
func (s *ConsoleService) SetGoodsStatus(ctx context.Context, req *consolev1.SetGoodsStatusRequest) (*consolev1.StatusReply, error) {
	a, err := s.operator(ctx)
	if err != nil {
		return nil, err
	}
	target := strconv.FormatInt(req.Id, 10)
	if err := s.uc.SetGoodsStatus(ctx, req.Id, req.Status); err != nil {
		s.auditWrite(ctx, a, "goods.status", target, "failure", err.Error())
		return nil, err
	}
	s.auditWrite(ctx, a, "goods.status", target, "success", strconv.FormatInt(int64(req.Status), 10))
	return &consolev1.StatusReply{Id: req.Id, Status: req.Status}, nil
}

// SetGoodsSkuStatus puts a goods SKU on or off shelf through the ops service.
func (s *ConsoleService) SetGoodsSkuStatus(ctx context.Context, req *consolev1.SetGoodsSkuStatusRequest) (*consolev1.StatusReply, error) {
	a, err := s.operator(ctx)
	if err != nil {
		return nil, err
	}
	target := strconv.FormatInt(req.Id, 10)
	if err := s.uc.SetGoodsSkuStatus(ctx, req.Id, req.Status); err != nil {
		s.auditWrite(ctx, a, "goods_sku.status", target, "failure", err.Error())
		return nil, err
	}
	s.auditWrite(ctx, a, "goods_sku.status", target, "success", strconv.FormatInt(int64(req.Status), 10))
	return &consolev1.StatusReply{Id: req.Id, Status: req.Status}, nil
}

// ReviewRefund approves or rejects a refund through the ops service.
func (s *ConsoleService) ReviewRefund(ctx context.Context, req *consolev1.ReviewRefundRequest) (*consolev1.RefundReview, error) {
	a, err := s.operator(ctx)
	if err != nil {
		return nil, err
	}
	target := strconv.FormatInt(req.Id, 10)
	review, err := s.uc.ReviewRefund(ctx, req.Id, req.Decision)
	if err != nil {
		s.auditWrite(ctx, a, "refund.review", target, "failure", err.Error())
		return nil, err
	}
	s.auditWrite(ctx, a, "refund.review", target, "success", req.Decision)
	return &consolev1.RefundReview{
		RefundId:     review.RefundID,
		RefundNo:     review.RefundNo,
		OrderId:      review.OrderID,
		Status:       review.Status,
		OrderStatus:  review.OrderStatus,
		RefundAmount: review.RefundAmount,
	}, nil
}