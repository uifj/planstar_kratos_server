package service

import (
	"context"
	"log/slog"

	"planstar/internal/biz"
)

// CatalogIndexer 是目录读模型的增量写入口，由 data.CatalogRepo 实现。
// 目录变更事件（来自运营后台或数据同步任务）通过 MQ 到达时触发增量更新。
type CatalogIndexer interface {
	UpsertActivity(ctx context.Context, activityID int64) error
	DeleteActivity(ctx context.Context, activityID int64) error
}

// EventHandler 消费领域事件：目前把订单/注册事件落成站内通知，并维护目录读模型。
// 作为 mq.Handler 的实现，service 层只依赖 biz 用例，不碰 amqp。
type EventHandler struct {
	messages *biz.MessageUsecase
	catalog  CatalogIndexer
	log      *slog.Logger
}

func NewEventHandler(messages *biz.MessageUsecase, catalog CatalogIndexer, log *slog.Logger) *EventHandler {
	return &EventHandler{messages: messages, catalog: catalog, log: log}
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
	case biz.EventCatalogActivityChanged:
		return h.upsertActivity(ctx, ev)
	case biz.EventCatalogActivityDeleted:
		return h.deleteActivity(ctx, ev)
	default:
		h.log.WarnContext(ctx, "unhandled domain event", "event", ev.Type)
		return nil
	}
}

// upsertActivity 处理目录更新事件；索引未启用时只记录日志，不影响事件消费。
func (h *EventHandler) upsertActivity(ctx context.Context, ev *biz.DomainEvent) error {
	if h.catalog == nil {
		return nil
	}
	activityID := eventActivityID(ev)
	if activityID <= 0 {
		h.log.WarnContext(ctx, "catalog event without activityId", "event", ev.Type)
		return nil
	}
	if err := h.catalog.UpsertActivity(ctx, activityID); err != nil {
		h.log.ErrorContext(ctx, "catalog incremental upsert failed", "activity_id", activityID, "err", err)
		return err
	}
	return nil
}

// deleteActivity 处理目录删除事件。
func (h *EventHandler) deleteActivity(ctx context.Context, ev *biz.DomainEvent) error {
	if h.catalog == nil {
		return nil
	}
	activityID := eventActivityID(ev)
	if activityID <= 0 {
		h.log.WarnContext(ctx, "catalog delete event without activityId", "event", ev.Type)
		return nil
	}
	if err := h.catalog.DeleteActivity(ctx, activityID); err != nil {
		h.log.ErrorContext(ctx, "catalog incremental delete failed", "activity_id", activityID, "err", err)
		return err
	}
	return nil
}

// eventActivityID 兼容 JSON 反序列化后的 float64 与直接构造时的整型。
func eventActivityID(ev *biz.DomainEvent) int64 {
	v, ok := ev.Data["activityId"]
	if !ok {
		v = ev.Data["activity_id"]
	}
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case float64:
		return int64(n)
	default:
		return 0
	}
}
