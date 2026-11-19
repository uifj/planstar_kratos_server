package biz

import "context"

// OpsRepo 是运营写操作仓储端口（社区动态 / 星币商城 / 退款审核），由 data 层实现。
// 运营写只经运维端口进入，业务公网 API 不暴露这些能力。
type OpsRepo interface {
	SetFeedStatus(ctx context.Context, id int64, status int32) error
	SetGoodsStatus(ctx context.Context, id int64, status int32) error
	SetGoodsSKUStatus(ctx context.Context, id int64, status int32) error
	ReviewRefund(ctx context.Context, id int64, approve bool) (*RefundReview, error)
}

// RefundReview 是退款审核结果（RefundAmount 单位为分）。
type RefundReview struct {
	RefundID     int64  `json:"refundId"`
	RefundNo     string `json:"refundNo"`
	OrderID      int64  `json:"orderId"`
	Status       string `json:"status"`
	OrderStatus  int32  `json:"orderStatus"`
	RefundAmount int64  `json:"refundAmount"`
}

// OpsUsecase 运营写用例：在边界校验入参后委托仓储，规则不下沉到 server。
type OpsUsecase struct {
	repo OpsRepo
}

func NewOpsUsecase(repo OpsRepo) *OpsUsecase { return &OpsUsecase{repo: repo} }

// SetFeedStatus 隐藏(0)/显示(1)一条社区动态。
func (uc *OpsUsecase) SetFeedStatus(ctx context.Context, id int64, status int32) error {
	if id <= 0 || (status != 0 && status != 1) {
		return ErrBadRequest
	}
	return uc.repo.SetFeedStatus(ctx, id, status)
}

// SetGoodsStatus 下架(0)/上架(1)一个星币商城商品。
func (uc *OpsUsecase) SetGoodsStatus(ctx context.Context, id int64, status int32) error {
	if id <= 0 || (status != 0 && status != 1) {
		return ErrBadRequest
	}
	return uc.repo.SetGoodsStatus(ctx, id, status)
}

// SetGoodsSKUStatus 下架(0)/上架(1)一个商品规格。
func (uc *OpsUsecase) SetGoodsSKUStatus(ctx context.Context, id int64, status int32) error {
	if id <= 0 || (status != 0 && status != 1) {
		return ErrBadRequest
	}
	return uc.repo.SetGoodsSKUStatus(ctx, id, status)
}

// ReviewRefund 审核退款单：approve 置 success 并回写订单退款状态，reject 置 rejected。
func (uc *OpsUsecase) ReviewRefund(ctx context.Context, id int64, approve bool) (*RefundReview, error) {
	if id <= 0 {
		return nil, ErrBadRequest
	}
	return uc.repo.ReviewRefund(ctx, id, approve)
}
