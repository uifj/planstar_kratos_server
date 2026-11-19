package shared

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"sync/atomic"
	"time"
)

// eventSeq 只在 crypto/rand 不可用时参与降级 ID 生成，避免全零或常量 ID。
var eventSeq uint64

// 领域事件类型（RabbitMQ routing key）。
const (
	EventUserRegistered         = "user.registered"
	EventOrderCreated           = "order.created"
	EventCatalogActivityChanged = "catalog.activity.changed"
	EventCatalogActivityDeleted = "catalog.activity.deleted"
)

// DomainEvent 是异步链路传递的纯领域事件（不依赖任何传输实现）。
type DomainEvent struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	UserID     int64          `json:"userId,omitempty"`
	OccurredAt time.Time      `json:"occurredAt"`
	Data       map[string]any `json:"data,omitempty"`
}

// EventPublisher 是事件发布端口，由 mq 适配器实现。
type EventPublisher interface {
	Publish(ctx context.Context, ev *DomainEvent) error
}

// NopEventPublisher 在未启用 MQ 时使用，保证用例无需判空。
type NopEventPublisher struct{}

func (NopEventPublisher) Publish(context.Context, *DomainEvent) error { return nil }

// NewEvent 组装一个带唯一 ID 的领域事件。
func NewEvent(eventType string, userID int64, data map[string]any) *DomainEvent {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// 事件 ID 用于消费端幂等；随机源失败时不能退化为全零或可预测常量。
		// 用纳秒时间 + 进程内自增序号生成降级 ID，仍保持唯一性。
		binary.BigEndian.PutUint64(b[:8], uint64(time.Now().UnixNano()))
		binary.BigEndian.PutUint64(b[8:], atomic.AddUint64(&eventSeq, 1))
	}
	return &DomainEvent{
		ID:         hex.EncodeToString(b),
		Type:       eventType,
		UserID:     userID,
		OccurredAt: time.Now(),
		Data:       data,
	}
}
