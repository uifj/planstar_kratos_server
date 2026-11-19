package biz

import (
	"context"
	"strings"
	"time"
)

// 支付单状态（对外字符串枚举）。
const (
	PaymentStatusPending = "pending"
	PaymentStatusSuccess = "success"
	PaymentStatusFailed  = "failed"
	PaymentStatusExpired = "expired"
)

// 支持的支付渠道。
const (
	PayChannelWechat   = "wechat"
	PayChannelAlipay   = "alipay"
	PayChannelUnionPay = "unionpay"
	PayChannelBalance  = "balance"
)

// DefaultPaymentTTL 是支付单默认有效期（30 分钟）。
const DefaultPaymentTTL = 30 * time.Minute

// DefaultRefundArrivalDays 是退款预计到账天数。
const DefaultRefundArrivalDays = 3

// maxRefundReasonText 是自定义退款原因的最大长度。
const maxRefundReasonText = 512

// payChannels 是支付渠道白名单。
var payChannels = map[string]bool{
	PayChannelWechat: true, PayChannelAlipay: true, PayChannelUnionPay: true, PayChannelBalance: true,
}

// refundReasonCodes 是退款原因码白名单（与 Flutter RefundReason 对齐）。
var refundReasonCodes = map[string]bool{
	"notNeeded": true, "boughtWrong": true, "tooExpensive": true, "foundBetter": true, "other": true,
}

// PayInput 是创建支付单入参。
type PayInput struct {
	OrderNumber string
	Channel     string
}

// PaymentOrder 是支付单（金额单位为分；时间字段为毫秒时间戳）。
type PaymentOrder struct {
	PayNo        string         `json:"payNo"`
	OrderNumber  string         `json:"orderNumber"`
	Amount       int64          `json:"amount"`
	Channel      string         `json:"channel"`
	Status       string         `json:"status"`
	PayParams    map[string]any `json:"payParams"`
	ExpireAt     int64          `json:"expireAt"`
	PaidAt       int64          `json:"paidAt"`
	ThirdTradeNo string         `json:"thirdTradeNo"`
}

// RefundInput 是退款申请入参。
type RefundInput struct {
	ReasonCode string
	ReasonText string
}

// Refund 是退款单（金额单位为分）。
type Refund struct {
	RefundNo            string `json:"refundNo"`
	OrderID             int64  `json:"orderId"`
	RefundableAmount    int64  `json:"refundableAmount"`
	Status              string `json:"status"`
	ExpectedArrivalDays int    `json:"expectedArrivalDays"`
}

// PaymentRepo 是支付/退款仓储端口。
type PaymentRepo interface {
	// CreatePayment 按（订单号 + 渠道）幂等创建支付单。
	CreatePayment(ctx context.Context, userID int64, in *PayInput, now time.Time) (*PaymentOrder, error)
	// PaymentDetail 按 payNo 或 orderNumber 查询支付单。
	PaymentDetail(ctx context.Context, userID int64, payNo, orderNumber string) (*PaymentOrder, error)
	// CreateRefund 按订单幂等创建退款单；订单状态不可退时返回 ErrRefundNotAllowed。
	CreateRefund(ctx context.Context, userID, orderID int64, in *RefundInput, now time.Time) (*Refund, error)
	RefundByOrder(ctx context.Context, userID, orderID int64) (*Refund, error)
}

// PaymentUsecase 支付与退款用例。
type PaymentUsecase struct {
	repo PaymentRepo
	now  func() time.Time
}

func NewPaymentUsecase(repo PaymentRepo) *PaymentUsecase {
	return &PaymentUsecase{repo: repo, now: time.Now}
}

// Pay 创建支付单；同订单同渠道重复调用返回同一支付单。
func (uc *PaymentUsecase) Pay(ctx context.Context, userID int64, in *PayInput) (*PaymentOrder, error) {
	if userID <= 0 {
		return nil, ErrUnauthorized
	}
	if in == nil {
		return nil, ErrBadRequest
	}
	in.OrderNumber = strings.TrimSpace(in.OrderNumber)
	in.Channel = strings.ToLower(strings.TrimSpace(in.Channel))
	if in.OrderNumber == "" || !payChannels[in.Channel] {
		return nil, ErrBadRequest
	}
	now := uc.now()
	p, err := uc.repo.CreatePayment(ctx, userID, in, now)
	if err != nil {
		return nil, err
	}
	normalizePaymentStatus(p, now)
	return p, nil
}

// PayDetail 查询支付单；已过 expireAt 仍为 pending 的支付单返回 expired。
func (uc *PaymentUsecase) PayDetail(ctx context.Context, userID int64, payNo, orderNumber string) (*PaymentOrder, error) {
	if userID <= 0 {
		return nil, ErrUnauthorized
	}
	payNo = strings.TrimSpace(payNo)
	orderNumber = strings.TrimSpace(orderNumber)
	if payNo == "" && orderNumber == "" {
		return nil, ErrBadRequest
	}
	p, err := uc.repo.PaymentDetail(ctx, userID, payNo, orderNumber)
	if err != nil {
		return nil, err
	}
	normalizePaymentStatus(p, uc.now())
	return p, nil
}

// Refund 申请退款；同订单重复提交返回同一退款单。
func (uc *PaymentUsecase) Refund(ctx context.Context, userID, orderID int64, in *RefundInput) (*Refund, error) {
	if userID <= 0 {
		return nil, ErrUnauthorized
	}
	if orderID <= 0 || in == nil {
		return nil, ErrBadRequest
	}
	in.ReasonCode = strings.TrimSpace(in.ReasonCode)
	if in.ReasonCode == "" {
		in.ReasonCode = "other"
	}
	if !refundReasonCodes[in.ReasonCode] {
		return nil, ErrBadRequest
	}
	in.ReasonText = strings.TrimSpace(in.ReasonText)
	if len([]rune(in.ReasonText)) > maxRefundReasonText {
		return nil, ErrBadRequest
	}
	return uc.repo.CreateRefund(ctx, userID, orderID, in, uc.now())
}

// RefundByOrder 查询订单的退款单。
func (uc *PaymentUsecase) RefundByOrder(ctx context.Context, userID, orderID int64) (*Refund, error) {
	if userID <= 0 {
		return nil, ErrUnauthorized
	}
	if orderID <= 0 {
		return nil, ErrBadRequest
	}
	return uc.repo.RefundByOrder(ctx, userID, orderID)
}

// normalizePaymentStatus 在读取时把已过期但仍为 pending 的支付单标记为 expired。
func normalizePaymentStatus(p *PaymentOrder, now time.Time) {
	if p == nil {
		return
	}
	if p.PayParams == nil {
		p.PayParams = map[string]any{}
	}
	if p.Status == PaymentStatusPending && p.ExpireAt > 0 && now.UnixMilli() >= p.ExpireAt {
		p.Status = PaymentStatusExpired
	}
}
