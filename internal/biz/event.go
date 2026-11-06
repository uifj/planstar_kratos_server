package biz

import "context"

// Event routing keys. exchanges are configured in conf.Mall.MQ.
const (
	EventOrderCreated        = "order.created"
	EventInventoryLockOK     = "inventory.lock.success"
	EventInventoryLockFailed = "inventory.lock.failed"
	EventPaymentSuccess      = "payment.success"
	EventOrderPaid           = "order.paid"
)

// Exchange names are the default topology names, kept here so biz code never
// has to import amqp or read config. The MQ adapter maps these to the
// configured exchange names.
const (
	ExchangeOrder     = "order.exchange"
	ExchangeInventory = "inventory.exchange"
	ExchangePayment   = "payment.exchange"
)

// EventItem is one line carried by an async event.
type EventItem struct {
	SkuID   string
	SkuName string
	Price   int64
	Num     int32
}

// DomainEvent is the DO that travels over RabbitMQ. It is intentionally free
// of proto/DTO and storage types: internal/mq encodes it as JSON and
// internal/service decodes it back into this same struct before calling a
// usecase. This keeps the DTO -> DO -> PO arrows from ever reversing.
type DomainEvent struct {
	ID         string      `json:"id"`
	Type       string      `json:"type"`
	OrderSn    string      `json:"order_sn"`
	PaymentNo  string      `json:"payment_no"`
	UserID     string      `json:"user_id"`
	Amount     int64       `json:"amount"`
	Reason     string      `json:"reason"`
	Items      []EventItem `json:"items"`
	OccurredAt int64       `json:"occurred_at"`
}

// EventPublisher is an outbound port implemented by internal/mq.
type EventPublisher interface {
	Publish(ctx context.Context, exchange, routingKey string, event *DomainEvent) error
}

// Inbound handler ports, implemented by internal/service. internal/mq only
// sees these interfaces; it never imports a usecase directly.
type OrderEventHandler interface {
	HandleInventoryLocked(ctx context.Context, event *DomainEvent) error
	HandleInventoryFailed(ctx context.Context, event *DomainEvent) error
	HandlePaymentSuccess(ctx context.Context, event *DomainEvent) error
}

type InventoryEventHandler interface {
	HandleOrderCreated(ctx context.Context, event *DomainEvent) error
	HandleOrderPaid(ctx context.Context, event *DomainEvent) error
}

type GoodsEventHandler interface {
	HandleOrderPaid(ctx context.Context, event *DomainEvent) error
}

// OrderTimeoutHandler is driven by internal/mq's timeout sweep. It is a
// separate port so the MQ server never depends on a concrete usecase.
type OrderTimeoutHandler interface {
	CancelTimedOut(ctx context.Context, before int64, limit int) (int, error)
}
