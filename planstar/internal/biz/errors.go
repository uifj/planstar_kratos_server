package biz

import "errors"

// 订单链路的领域错误（service 层据 errors.Is 映射 HTTP 状态码与业务错误码）。
var (
	// ErrPriceChanged 客户端金额/报价版本与服务端权威计算结果不一致（409 PRICE_CHANGED）。
	ErrPriceChanged = errors.New("price changed")
	// ErrSoldOut 库存不足或演出已售罄（409 SOLD_OUT）。
	ErrSoldOut = errors.New("sold out")
	// ErrLimitExceeded 购买数量超过限购规则（409 LIMIT_EXCEEDED）。
	ErrLimitExceeded = errors.New("limit exceeded")
	// ErrQuoteExpired 报价快照不存在或已过期（409 QUOTE_EXPIRED）。
	ErrQuoteExpired = errors.New("quote expired")
)
