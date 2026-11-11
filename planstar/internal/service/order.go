package service

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"planstar/internal/biz"
	"planstar/internal/pkg/httpx"
)

func (s *Service) orders(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	items, err := s.order.Orders(r.Context(), userID,
		httpx.IntQuery(r, "status", 0),
		httpx.IntQuery(r, "orderType", 0),
		httpx.IntQuery(r, "pageIndex", 1),
		httpx.IntQuery(r, "pageSize", 20))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, items)
}

func (s *Service) orderDetail(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	item, err := s.order.OrderByNumber(r.Context(), userID, r.URL.Query().Get("orderNumber"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

// confirmOrder 服务端权威报价：按企划/线下回报场次/回报档位/数量重新计算金额，返回带版本号与过期时间的快照。
func (s *Service) confirmOrder(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	body, err := httpx.ReadMap(r)
	if err != nil {
		s.fail(w, r, biz.ErrBadRequest)
		return
	}
	in := &biz.QuoteInput{
		ActivityID:       httpx.Int64Any(body, "activityId", "activity_id", "showId"),
		ActivityEventID:  httpx.Int64Any(body, "activityEventId", "activity_event_id", "sessionId"),
		TicketCategoryID: httpx.Int64Any(body, "ticketCategoryId", "ticket_category_id", "ticketId"),
		Quantity:         int(httpx.Int64Any(body, "quantity", "count", "buyCount")),
	}
	quote, err := s.order.Quote(r.Context(), userID, in)
	if err != nil {
		orderConfirmTotal.WithLabelValues("failure").Inc()
		s.fail(w, r, err)
		return
	}
	orderConfirmTotal.WithLabelValues("success").Inc()
	httpx.JSON(w, http.StatusOK, quote)
}

// makeOrder 下单：携带 quoteId（推荐）或 amount，服务端按快照重算并校验后创建订单。
// 请求头 Idempotency-Key 用于幂等，同 key 重复请求返回首次创建的订单。
func (s *Service) makeOrder(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	body, err := httpx.ReadMap(r)
	if err != nil {
		s.fail(w, r, biz.ErrBadRequest)
		return
	}
	amount := httpx.Int64Any(body, "amount", "payAmount", "pay_amount")
	in := &biz.OrderInput{
		ActivityID:       httpx.Int64Any(body, "activityId", "activity_id", "showId"),
		ActivityEventID:  httpx.Int64Any(body, "activityEventId", "activity_event_id", "sessionId"),
		TicketCategoryID: httpx.Int64Any(body, "ticketCategoryId", "ticket_category_id", "ticketId"),
		OrderType:        int(httpx.Int64Any(body, "orderType", "order_type")),
		Quantity:         int(httpx.Int64Any(body, "quantity", "count", "buyCount")),
		Amount:           amount,
		PayAmount:        httpx.Int64Any(body, "payAmount", "pay_amount"),
		TotalAmount:      httpx.Int64Any(body, "totalAmount", "total_amount"),
		QuoteID:          httpx.StrAny(body, "quoteId", "quote_id"),
		QuoteVersion:     int(httpx.Int64Any(body, "quoteVersion", "quote_version", "version")),
		IdempotencyKey:   strings.TrimSpace(r.Header.Get("Idempotency-Key")),
		Name:             httpx.StrAny(body, "name", "consignee"),
		Phone:            httpx.StrAny(body, "phone", "mobile"),
		AddressID:        httpx.Int64Any(body, "userAddressId", "user_address_id", "addressId"),
	}
	// 幂等键是服务端约束，不能依赖客户端自觉携带。
	// 客户端未带时按「用户 + 报价快照 + 订单要素」派生稳定键，
	// 双击 / 网关重试 / 504 重放都会命中同一订单，不会重复扣库存。
	if in.IdempotencyKey == "" {
		in.IdempotencyKey = deriveIdempotencyKey(userID, in)
	}
	item, err := s.order.MakeOrder(r.Context(), userID, in)
	if err != nil {
		orderCreatedTotal.WithLabelValues(orderResultLabel(err)).Inc()
		s.fail(w, r, err)
		return
	}
	orderCreatedTotal.WithLabelValues("success").Inc()
	httpx.JSON(w, http.StatusOK, item)
}

// deriveIdempotencyKey 生成服务端兜底幂等键（auto: + 16 字节摘要，长度 37 < VARCHAR(128)）。
// 只包含影响订单实质的字段，避免把无关请求头变化算成新订单。
func deriveIdempotencyKey(userID int64, in *biz.OrderInput) string {
	raw := fmt.Sprintf("%d|%s|%d|%d|%d|%d|%d|%d|%d|%d|%d|%s|%s|%d",
		userID, in.QuoteID, in.QuoteVersion, in.ActivityID, in.ActivityEventID,
		in.TicketCategoryID, in.OrderType, in.Quantity, in.Amount, in.PayAmount,
		in.TotalAmount, in.Name, in.Phone, in.AddressID)
	sum := sha256.Sum256([]byte(raw))
	return "auto:" + hex.EncodeToString(sum[:16])
}

// orderResultLabel 把下单错误归类为稳定的指标标签，避免标签基数膨胀。
func orderResultLabel(err error) string {
	switch {
	case errors.Is(err, biz.ErrPriceChanged):
		return "price_changed"
	case errors.Is(err, biz.ErrSoldOut):
		return "sold_out"
	case errors.Is(err, biz.ErrLimitExceeded):
		return "limit_exceeded"
	case errors.Is(err, biz.ErrQuoteExpired):
		return "quote_expired"
	default:
		return "error"
	}
}
