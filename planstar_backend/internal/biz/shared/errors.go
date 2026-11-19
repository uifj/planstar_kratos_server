// Package shared 是领域层的共享内核：跨域共用的哨兵错误、事件端口、单位/时间换算与脱敏助手。
//
// 依赖方向严格单向：biz/<domain> → biz/shared → 标准库。
// shared 不得 import 任何 biz 子包、data、service、server 或基础设施 SDK。
package shared

import (
	"errors"
	"fmt"
	"strings"
)

// 通用领域错误（service.fail 依据 errors.Is 映射 HTTP 状态与错误码）。
var (
	ErrBadRequest   = errors.New("bad request")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrRateLimited  = errors.New("rate limited")
	ErrLoginFailed  = errors.New("login failed")
)

// 订单与支付链路错误（主站 409；REFUND_NOT_ALLOWED 在 9108 运维面为 400）。
var (
	ErrPriceChanged      = errors.New("price changed")
	ErrSoldOut           = errors.New("sold out")
	ErrLimitExceeded     = errors.New("limit exceeded")
	ErrQuoteExpired      = errors.New("quote expired")
	ErrRefundNotAllowed  = errors.New("refund not allowed")
	ErrInsufficientCoins = errors.New("insufficient coins")
)

// 企划创意域错误。
var (
	ErrIdeaNotFound       = errors.New("idea not found")
	ErrIdeaSelfVote       = errors.New("idea self vote forbidden")
	ErrIdeaInvalidTier    = errors.New("idea invalid coin tier")
	ErrIdeaReasonRequired = errors.New("idea reason required")
	ErrIdeaInvalidReason  = errors.New("idea invalid reason code")
)

// BlacklistHit 是一条黑名单命中项（rule=规则分类，keyword=命中词或缺失项）。
type BlacklistHit struct {
	Rule    string `json:"rule"`
	Keyword string `json:"keyword"`
}

// BlacklistBlockedError 表示企划命中黑名单，禁止通过审核（映射 400，并携带原因）。
type BlacklistBlockedError struct {
	Hits []BlacklistHit
}

func (e *BlacklistBlockedError) Error() string {
	parts := make([]string, 0, len(e.Hits))
	for _, hit := range e.Hits {
		parts = append(parts, fmt.Sprintf("%s(%s)", hit.Rule, hit.Keyword))
	}
	return "企划命中品类黑名单：" + strings.Join(parts, "、")
}

// Unwrap 让调用方可按 ErrBadRequest 统一映射 HTTP 400。
func (e *BlacklistBlockedError) Unwrap() error { return ErrBadRequest }
