package biz

import "planstar/internal/biz/shared"

// 兼容层（ADR-0001 P1）：订单链路错误定义已下沉到 biz/shared。
// P2 层内按域分包时，各域直接引用 shared，本文件随之删除。
// errors.Is 仍然成立：这里是同一个 error 值的别名，不是新建错误。
var (
	ErrPriceChanged      = shared.ErrPriceChanged
	ErrSoldOut           = shared.ErrSoldOut
	ErrLimitExceeded     = shared.ErrLimitExceeded
	ErrQuoteExpired      = shared.ErrQuoteExpired
	ErrRefundNotAllowed  = shared.ErrRefundNotAllowed
	ErrInsufficientCoins = shared.ErrInsufficientCoins
)
