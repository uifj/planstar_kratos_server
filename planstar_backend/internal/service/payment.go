package service

import (
	"net/http"

	"planstar/internal/biz"
	"planstar/internal/pkg/httpx"
)

// pay 创建支付单（同订单同渠道幂等）。
func (s *Service) pay(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	body, err := httpx.ReadMap(r)
	if err != nil {
		s.fail(w, r, biz.ErrBadRequest)
		return
	}
	in := &biz.PayInput{
		OrderNumber: httpx.StrAny(body, "orderNumber", "order_number"),
		Channel:     httpx.StrAny(body, "channel", "payChannel", "pay_channel"),
	}
	item, err := s.payment.Pay(r.Context(), userID, in)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

// payDetail 按 payNo 或 orderNumber 查询支付单状态。
func (s *Service) payDetail(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	item, err := s.payment.PayDetail(r.Context(), userID,
		r.URL.Query().Get("payNo"), r.URL.Query().Get("orderNumber"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

// createRefund 申请退款（同订单幂等；不可退返回 409 REFUND_NOT_ALLOWED）。
func (s *Service) createRefund(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	orderID, ok := pathID(r, "id")
	if !ok {
		s.fail(w, r, biz.ErrBadRequest)
		return
	}
	body, err := httpx.ReadMap(r)
	if err != nil {
		s.fail(w, r, biz.ErrBadRequest)
		return
	}
	in := &biz.RefundInput{
		ReasonCode: httpx.StrAny(body, "reasonCode", "reason_code"),
		ReasonText: httpx.StrAny(body, "reasonText", "reason_text", "reason"),
	}
	item, err := s.payment.Refund(r.Context(), userID, orderID, in)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}

// orderRefund 查询订单的退款单。
func (s *Service) orderRefund(w http.ResponseWriter, r *http.Request) {
	userID, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	orderID, ok := pathID(r, "id")
	if !ok {
		s.fail(w, r, biz.ErrBadRequest)
		return
	}
	item, err := s.payment.RefundByOrder(r.Context(), userID, orderID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, item)
}
