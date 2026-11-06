package biz

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"
)

// 领域事件类型（RabbitMQ routing key）。
const (
	EventUserRegistered = "user.registered"
	EventOrderCreated   = "order.created"
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
	_, _ = rand.Read(b)
	return &DomainEvent{
		ID:         hex.EncodeToString(b),
		Type:       eventType,
		UserID:     userID,
		OccurredAt: time.Now(),
		Data:       data,
	}
}
