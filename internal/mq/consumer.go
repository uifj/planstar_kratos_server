package mq

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/go-kratos/kratos-layout/internal/biz"
	"github.com/go-kratos/kratos/v3/log"
	amqp "github.com/rabbitmq/amqp091-go"
)

type consumerSpec struct {
	queue  string
	name   string
	handle func(context.Context, *biz.DomainEvent) error
}

type consumers struct {
	state    *state
	cfg      Config
	consumed biz.ConsumedEventRepo
	specs    []consumerSpec
}

func newConsumers(
	s *state,
	cfg Config,
	consumed biz.ConsumedEventRepo,
	orderEvents biz.OrderEventHandler,
	inventoryEvents biz.InventoryEventHandler,
	goodsEvents biz.GoodsEventHandler,
) *consumers {
	specs := []consumerSpec{
		{
			queue:  cfg.OrderCreatedQueue,
			name:   "mall.order.created",
			handle: inventoryEvents.HandleOrderCreated,
		},
		{
			queue: cfg.InventoryResultQueue,
			name:  "mall.inventory.result",
			handle: func(ctx context.Context, e *biz.DomainEvent) error {
				switch e.Type {
				case biz.EventInventoryLockOK:
					return orderEvents.HandleInventoryLocked(ctx, e)
				case biz.EventInventoryLockFailed:
					return orderEvents.HandleInventoryFailed(ctx, e)
				default:
					return nil
				}
			},
		},
		{
			queue:  cfg.PaymentSuccessQueue,
			name:   "mall.payment.success",
			handle: orderEvents.HandlePaymentSuccess,
		},
		{
			queue: cfg.OrderPaidGoodsQueue,
			name:  "mall.order.paid.goods",
			handle: func(ctx context.Context, e *biz.DomainEvent) error {
				// Confirm stock deduction before touching sold_num / ES.
				if err := inventoryEvents.HandleOrderPaid(ctx, e); err != nil {
					return err
				}
				return goodsEvents.HandleOrderPaid(ctx, e)
			},
		},
	}
	return &consumers{state: s, cfg: cfg, consumed: consumed, specs: specs}
}

func (c *consumers) run(ctx context.Context) {
	var wg sync.WaitGroup
	for _, spec := range c.specs {
		ch, err := c.state.consumerChannel()
		if err != nil {
			log.Error("mq: failed opening consumer channel", "queue", spec.queue, "err", err)
			continue
		}
		if err := ch.Qos(1, 0, false); err != nil {
			_ = ch.Close()
			log.Error("mq: failed setting qos", "queue", spec.queue, "err", err)
			continue
		}
		deliveries, err := ch.ConsumeWithContext(ctx, spec.queue, spec.name,
			false, false, false, false, nil)
		if err != nil {
			_ = ch.Close()
			log.Error("mq: failed starting consumer", "queue", spec.queue, "err", err)
			continue
		}
		wg.Add(1)
		go func(ch *amqp.Channel, spec consumerSpec, deliveries <-chan amqp.Delivery) {
			defer wg.Done()
			defer ch.Close()
			c.loop(ctx, spec, ch, deliveries)
		}(ch, spec, deliveries)
	}
	wg.Wait()
}

func (c *consumers) loop(ctx context.Context, spec consumerSpec, ch *amqp.Channel, deliveries <-chan amqp.Delivery) {
	for {
		select {
		case <-ctx.Done():
			return
		case d, ok := <-deliveries:
			if !ok {
				return
			}
			c.handleDelivery(ctx, spec, ch, d)
		}
	}
}

func (c *consumers) handleDelivery(ctx context.Context, spec consumerSpec, ch *amqp.Channel, d amqp.Delivery) {
	var event biz.DomainEvent
	if err := json.Unmarshal(d.Body, &event); err != nil {
		// A malformed payload can never be fixed by retrying: discard it.
		log.Warn("mq: dropping malformed event", "queue", spec.queue, "err", err)
		_ = d.Ack(false)
		return
	}
	if event.ID == "" {
		event.ID = d.MessageId
	}
	if exists, err := c.consumed.Exists(ctx, event.ID, spec.name); err == nil && exists {
		_ = d.Ack(false)
		return
	}
	if err := spec.handle(ctx, &event); err != nil {
		log.Warn("mq: event handler failed", "queue", spec.queue, "type", event.Type, "err", err)
		c.retryOrDeadLetter(ctx, ch, d)
		return
	}
	if err := c.consumed.Mark(ctx, event.ID, spec.name); err != nil {
		log.Warn("mq: failed marking consumed event", "event", event.ID, "err", err)
	}
	_ = d.Ack(false)
}

// retryOrDeadLetter republishes with an incremented x-retry-count header and
// acks the original. Once max_retry is reached the message is rejected
// without requeue, which routes it to the DLX/DLQ declared in topology.
func (c *consumers) retryOrDeadLetter(ctx context.Context, ch *amqp.Channel, d amqp.Delivery) {
	retry := retryCount(d.Headers)
	if retry+1 >= c.cfg.MaxRetry {
		_ = d.Nack(false, false)
		return
	}
	headers := cloneHeaders(d.Headers)
	headers["x-retry-count"] = int32(retry + 1)
	err := ch.PublishWithContext(ctx, d.Exchange, d.RoutingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		MessageId:    d.MessageId,
		Headers:      headers,
		Body:         d.Body,
	})
	if err != nil {
		_ = d.Nack(false, true)
		return
	}
	_ = d.Ack(false)
}

func retryCount(h amqp.Table) int {
	if h == nil {
		return 0
	}
	switch v := h["x-retry-count"].(type) {
	case int:
		return v
	case int8:
		return int(v)
	case int16:
		return int(v)
	case int32:
		return int(v)
	case int64:
		return int(v)
	case uint8:
		return int(v)
	case uint16:
		return int(v)
	case uint32:
		return int(v)
	case uint64:
		return int(v)
	default:
		return 0
	}
}

func cloneHeaders(h amqp.Table) amqp.Table {
	out := make(amqp.Table, len(h)+1)
	for k, v := range h {
		out[k] = v
	}
	return out
}
