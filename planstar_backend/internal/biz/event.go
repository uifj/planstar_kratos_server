package biz

import "planstar/internal/biz/shared"

// 兼容层（ADR-0001 P1）：事件类型、事件结构与发布端口已下沉到 biz/shared。
// P2 按域分包后，mq 适配器与各域直接引用 shared，本文件随之删除。
const (
	EventUserRegistered         = shared.EventUserRegistered
	EventOrderCreated           = shared.EventOrderCreated
	EventCatalogActivityChanged = shared.EventCatalogActivityChanged
	EventCatalogActivityDeleted = shared.EventCatalogActivityDeleted
)

type DomainEvent = shared.DomainEvent
type EventPublisher = shared.EventPublisher
type NopEventPublisher = shared.NopEventPublisher

func NewEvent(eventType string, userID int64, data map[string]any) *DomainEvent {
	return shared.NewEvent(eventType, userID, data)
}
