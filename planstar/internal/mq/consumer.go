package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"planstar/internal/biz"
)

const (
	MaxRetry       = 5
	RetryHeaderKey = "x-retry-count"
)

// Handler 处理已解码的领域事件；返回错误会触发重试 / 死信。
type Handler func(context.Context, *biz.DomainEvent) error

// Consumer 订阅一个队列（可绑定多个 routing key），带确认、重试与死信。
type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   string
	dlx     string
	handler Handler
}

func NewConsumer(addr, exchange, queue string, routingKeys []string, handler Handler) (*Consumer, error) {
	conn, err := amqp.DialConfig(addr, amqp.Config{Dial: amqp.DefaultDial(3 * time.Second)})
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	cleanup := func() {
		_ = ch.Close()
		_ = conn.Close()
	}
	if err := ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		cleanup()
		return nil, err
	}
	dlx := exchange + ".dlx"
	if err := ch.ExchangeDeclare(dlx, "topic", true, false, false, false, nil); err != nil {
		cleanup()
		return nil, err
	}
	q, err := ch.QueueDeclare(queue, true, false, false, false, nil)
	if err != nil {
		cleanup()
		return nil, err
	}
	for _, key := range routingKeys {
		if err := ch.QueueBind(q.Name, key, exchange, false, nil); err != nil {
			cleanup()
			return nil, err
		}
	}
	// 死信队列：保证超过重试次数的消息可追踪、可重放。
	if _, err := ch.QueueDeclare(queue+".dlq", true, false, false, false, nil); err != nil {
		cleanup()
		return nil, err
	}
	if err := ch.QueueBind(queue+".dlq", "failed."+queue, dlx, false, nil); err != nil {
		cleanup()
		return nil, err
	}
	return &Consumer{conn: conn, channel: ch, queue: q.Name, dlx: dlx, handler: handler}, nil
}

func (c *Consumer) Run(ctx context.Context) error {
	msgs, err := c.channel.Consume(c.queue, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case d, ok := <-msgs:
			if !ok {
				return nil
			}
			if err := c.handle(ctx, d); err != nil {
				c.retryOrDLQ(ctx, d, err)
				continue
			}
			_ = d.Ack(false)
		}
	}
}

func (c *Consumer) handle(ctx context.Context, d amqp.Delivery) error {
	msgCtx := extractTrace(ctx, d.Headers)
	msgCtx, span := consumeSpan(msgCtx, c.queue, d.RoutingKey)
	defer span.End()

	var ev biz.DomainEvent
	if err := json.Unmarshal(d.Body, &ev); err != nil {
		span.RecordError(err)
		return fmt.Errorf("decode event: %w", err)
	}
	if err := c.handler(msgCtx, &ev); err != nil {
		span.RecordError(err)
		return err
	}
	return nil
}

func (c *Consumer) retryOrDLQ(ctx context.Context, d amqp.Delivery, err error) {
	attempts := deliveryRetryCount(d.Headers)
	if attempts+1 >= MaxRetry {
		_ = c.channel.Publish(
			// 死信 exchange 由 NewConsumer 保证存在。
			c.queueExchangeDLX(),
			"failed."+c.queue,
			false,
			false,
			amqp.Publishing{
				ContentType:  d.ContentType,
				DeliveryMode: amqp.Persistent,
				Headers:      d.Headers,
				Body:         d.Body,
			},
		)
		_ = d.Ack(false)
		return
	}

	headers := make(amqp.Table, len(d.Headers)+1)
	for k, v := range d.Headers {
		headers[k] = v
	}
	headers[RetryHeaderKey] = attempts + 1
	delay := time.Duration(attempts+1) * time.Second
	select {
	case <-ctx.Done():
	case <-time.After(delay):
	}
	_ = c.channel.Publish(
		"",
		c.queue,
		false,
		false,
		amqp.Publishing{
			ContentType:  d.ContentType,
			DeliveryMode: amqp.Persistent,
			Headers:      headers,
			Body:         d.Body,
		},
	)
	_ = d.Ack(false)
}

// queueExchangeDLX 返回死信 exchange 名（与 NewConsumer 中的声明保持一致）。
func (c *Consumer) queueExchangeDLX() string {
	return c.dlx
}

func deliveryRetryCount(headers amqp.Table) int {
	if headers == nil {
		return 0
	}
	if v, ok := headers[RetryHeaderKey]; ok {
		switch n := v.(type) {
		case int32:
			return int(n)
		case int64:
			return int(n)
		case int:
			return n
		}
	}
	return 0
}

func (c *Consumer) Close() {
	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
}
