// Package errcode 是全站 HTTP 错误码的单一事实源。
//
// service.fail()、server 的运维面/中间件/鉴权错误体都必须引用本包常量，
// 禁止再散落字符串字面量。docs/api/API_CONTRACT.md 的码表由
// internal/governance 的契约测试双向校验，改动本文件必须同步契约文档。
package errcode

const (
	// 通用状态码
	BadRequest         = "BAD_REQUEST"
	Unauthorized       = "UNAUTHORIZED"
	Forbidden          = "FORBIDDEN"
	NotFound           = "NOT_FOUND"
	Conflict           = "CONFLICT"
	RateLimited        = "RATE_LIMITED"
	PayloadTooLarge    = "PAYLOAD_TOO_LARGE"
	Internal           = "INTERNAL"
	Timeout            = "TIMEOUT"
	ServiceUnavailable = "SERVICE_UNAVAILABLE"

	// 订单与支付链路（主站 409；REFUND_NOT_ALLOWED 在 9108 运维面为 400）
	PriceChanged     = "PRICE_CHANGED"
	SoldOut          = "SOLD_OUT"
	LimitExceeded    = "LIMIT_EXCEEDED"
	QuoteExpired     = "QUOTE_EXPIRED"
	RefundNotAllowed = "REFUND_NOT_ALLOWED"

	// 钱包 / 商城
	InsufficientCoins = "INSUFFICIENT_COINS"

	// 企划创意投票
	IdeaNotFound          = "IDEA_NOT_FOUND"
	IdeaSelfVoteForbidden = "IDEA_SELF_VOTE_FORBIDDEN"
	InvalidCoinTier       = "INVALID_COIN_TIER"
	ReasonRequired        = "REASON_REQUIRED"
	InvalidReasonCode     = "INVALID_REASON_CODE"
)

// All 返回全部错误码，顺序稳定，供契约一致性测试使用。
func All() []string {
	return []string{
		BadRequest, Unauthorized, Forbidden, NotFound, Conflict,
		RateLimited, PayloadTooLarge, Internal, Timeout, ServiceUnavailable,
		PriceChanged, SoldOut, LimitExceeded, QuoteExpired, RefundNotAllowed,
		InsufficientCoins,
		IdeaNotFound, IdeaSelfVoteForbidden, InvalidCoinTier, ReasonRequired, InvalidReasonCode,
	}
}
