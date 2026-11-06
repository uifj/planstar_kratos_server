package service

import (
	"net/http"

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

// confirmOrder 客户端原样透传确认体，后端校验登录后回显（以 order_confirm_snapshot 为蓝本后续再固化为强类型）。
func (s *Service) confirmOrder(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireUser(w, r); !ok {
		return
	}
	body, err := httpx.ReadMap(r)
	if err != nil {
		s.fail(w, r, biz.ErrBadRequest)
		return
	}
	httpx.JSON(w, http.StatusOK, body)
}

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
	in := &biz.OrderInput{
		ActivityID:      httpx.Int64Any(body, "activityId", "activity_id", "showId"),
		ActivityEventID: httpx.Int64Any(body, "activityEventId", "activity_event_id", "sessionId"),
		OrderType:       int(httpx.Int64Any(body, "orderType", "order_type")),
		PayAmount:       httpx.Int64Any(body, "payAmount", "pay_amount"),
		TotalAmount:     httpx.Int64Any(body, "totalAmount", "total_amount"),
		Name:            httpx.StrAny(body, "name", "consignee"),
		Phone:           httpx.StrAny(body, "phone", "mobile"),
		AddressID:       httpx.Int64Any(body, "userAddressId", "user_address_id", "addressId"),
	}
	item, err := s.order.MakeOrder(r.Context(), userID, in)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}
