package service

import (
	"context"
	"log/slog"

	"planstar/internal/biz"
)

// EventHandler 消费领域事件：目前把订单/注册事件落成站内通知。
// 作为 mq.Handler 的实现，service 层只依赖 biz 用例，不碰 amqp。
type EventHandler struct {
	messages *biz.MessageUsecase
	log      *slog.Logger
}

func NewEventHandler(messages *biz.MessageUsecase, log *slog.Logger) *EventHandler {
	return &EventHandler{messages: messages, log: log}
}

func (h *EventHandler) Handle(ctx context.Context, ev *biz.DomainEvent) error {
	switch ev.Type {
	case biz.EventUserRegistered:
		mobile, _ := ev.Data["mobile"].(string)
		if len(mobile) >= 4 {
			mobile = "****" + mobile[len(mobile)-4:]
		}
		return h.messages.Notify(ctx, ev.UserID, "欢迎加入企划星",
			"你的账号（"+mobile+"）已注册成功，去发现感兴趣的活动吧。")
	case biz.EventOrderCreated:
		orderNumber, _ := ev.Data["orderNumber"].(string)
		return h.messages.Notify(ctx, ev.UserID, "订单创建成功",
			"订单 "+orderNumber+" 已创建，请尽快完成支付。")
	default:
		h.log.WarnContext(ctx, "unhandled domain event", "event", ev.Type)
		return nil
	}
}
