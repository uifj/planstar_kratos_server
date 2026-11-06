package mq

import (
	"time"

	"github.com/go-kratos/kratos-layout/internal/biz"
	"github.com/go-kratos/kratos-layout/internal/conf"
)

const (
	defaultMQAddr            = "amqp://guest:guest@127.0.0.1:5672/"
	defaultDLXExchange       = "mall.dlx"
	defaultDLQ               = "q.mall.dlx"
	defaultOrderCreatedQueue = "q.order.created"
	defaultInventoryQueue    = "q.inventory.result"
	defaultPaymentQueue      = "q.payment.success"
	defaultOrderPaidQueue    = "q.order.paid.goods"
	defaultMaxRetry          = 5
	defaultOutboxInterval    = time.Second
	defaultPayTimeout        = 15 * time.Minute
	defaultTimeoutInterval   = 30 * time.Second
	defaultOutboxBatch       = 50
)

// Config is the fully-resolved MQ topology and timing configuration. It is
// built once from conf.Mall so no other file reads protobuf config getters.
type Config struct {
	Addr                 string
	OrderExchange        string
	InventoryExchange    string
	PaymentExchange      string
	DLXExchange          string
	DLQ                  string
	OrderCreatedQueue    string
	InventoryResultQueue string
	PaymentSuccessQueue  string
	OrderPaidGoodsQueue  string
	MaxRetry             int
	OutboxScanInterval   time.Duration
	OrderPayTimeout      time.Duration
	TimeoutLoopInterval  time.Duration
	OutboxBatchSize      int
}

func NewConfig(m *conf.Mall) Config {
	c := Config{
		Addr:                 defaultMQAddr,
		OrderExchange:        biz.ExchangeOrder,
		InventoryExchange:    biz.ExchangeInventory,
		PaymentExchange:      biz.ExchangePayment,
		DLXExchange:          defaultDLXExchange,
		DLQ:                  defaultDLQ,
		OrderCreatedQueue:    defaultOrderCreatedQueue,
		InventoryResultQueue: defaultInventoryQueue,
		PaymentSuccessQueue:  defaultPaymentQueue,
		OrderPaidGoodsQueue:  defaultOrderPaidQueue,
		MaxRetry:             defaultMaxRetry,
		OutboxScanInterval:   defaultOutboxInterval,
		OrderPayTimeout:      defaultPayTimeout,
		TimeoutLoopInterval:  defaultTimeoutInterval,
		OutboxBatchSize:      defaultOutboxBatch,
	}
	if m == nil {
		return c
	}
	if q := m.GetMq(); q != nil {
		if v := q.GetAddr(); v != "" {
			c.Addr = v
		}
		if v := q.GetOrderExchange(); v != "" {
			c.OrderExchange = v
		}
		if v := q.GetInventoryExchange(); v != "" {
			c.InventoryExchange = v
		}
		if v := q.GetPaymentExchange(); v != "" {
			c.PaymentExchange = v
		}
		if v := q.GetDlxExchange(); v != "" {
			c.DLXExchange = v
		}
		if v := q.GetOrderCreatedQueue(); v != "" {
			c.OrderCreatedQueue = v
		}
		if v := q.GetInventoryResultQueue(); v != "" {
			c.InventoryResultQueue = v
		}
		if v := q.GetPaymentSuccessQueue(); v != "" {
			c.PaymentSuccessQueue = v
		}
		if v := q.GetOrderPaidGoodsQueue(); v != "" {
			c.OrderPaidGoodsQueue = v
		}
		if v := q.GetMaxRetry(); v > 0 {
			c.MaxRetry = int(v)
		}
	}
	if b := m.GetBusiness(); b != nil {
		if d := b.GetOutboxScanInterval(); d != nil && d.AsDuration() > 0 {
			c.OutboxScanInterval = d.AsDuration()
		}
		if d := b.GetOrderPayTimeout(); d != nil && d.AsDuration() > 0 {
			c.OrderPayTimeout = d.AsDuration()
		}
		if d := b.GetTimeoutLoopInterval(); d != nil && d.AsDuration() > 0 {
			c.TimeoutLoopInterval = d.AsDuration()
		}
		if v := b.GetOutboxBatchSize(); v > 0 {
			c.OutboxBatchSize = int(v)
		}
	}
	return c
}

// exchange maps a logical exchange name (the constants declared in biz) to
// the configured physical exchange name. biz code therefore stays unaware of
// deployment-specific names.
func (c Config) exchange(logical string) string {
	switch logical {
	case biz.ExchangeOrder:
		return c.OrderExchange
	case biz.ExchangeInventory:
		return c.InventoryExchange
	case biz.ExchangePayment:
		return c.PaymentExchange
	default:
		return logical
	}
}

// exchangeForEventType routes an outbox row to its logical exchange. Failed
// events whose type is unknown are marked failed by the relay.
func exchangeForEventType(eventType string) string {
	switch eventType {
	case biz.EventOrderCreated:
		return biz.ExchangeOrder
	case biz.EventPaymentSuccess, biz.EventOrderPaid:
		return biz.ExchangePayment
	case biz.EventInventoryLockOK, biz.EventInventoryLockFailed:
		return biz.ExchangeInventory
	default:
		return ""
	}
}
