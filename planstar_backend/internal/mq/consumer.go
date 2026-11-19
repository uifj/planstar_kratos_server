package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"planstar/internal/biz"
)

const (
	MaxRetry       = 5
	RetryHeaderKey = "x-retry-count"
	// prefetchCount 限制未确认消息数，避免 broker 无限推送打满消费者内存。
	prefetchCount = 32
	// retryDelayBase 是重试指数退避的基数（1s/2s/4s/8s/16s）。
	retryDelayBase = time.Second
)

// Handler 处理已解码的领域事件；返回错误会触发重试 / 死信。
type Handler func(context.Context, *biz.DomainEvent) error

// Consumer 订阅一个队列（可绑定多个 routing key），带确认、重试与 broker 级死信。
type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   string
	dlx     string
	handler Handler
	log     *slog.Logger
}

func NewConsumer(addr, exchange, queue string, routingKeys []string, handler Handler) (*Consumer, error) {
	return NewConsumerWithLogger(addr, exchange, queue, routingKeys, handler, slog.Default())
}

func NewConsumerWithLogger(addr, exchange, queue string, routingKeys []string, handler Handler, log *slog.Logger) (*Consumer, error) {
	if log == nil {
		log = slog.Default()
	}
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
	// 未确认消息上限：避免单消费者无界缓存导致 OOM。
	if err := ch.Qos(prefetchCount, 0, false); err != nil {
		cleanup()
		return nil, err
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
	// 主队列声明 broker 级 DLX：reject/Nack(requeue=false) 的消息自动进 DLQ，
	// 不再依赖「手工 Publish 后无条件 Ack」这种可能丢消息的做法。
	dlqRoutingKey := "failed." + queue
	q, err := ch.QueueDeclare(queue, true, false, false, false, amqp.Table{
		"x-dead-letter-exchange":    dlx,
		"x-dead-letter-routing-key": dlqRoutingKey,
	})
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
	if _, err := ch.QueueDeclare(queue+".dlq", true, false, false, false, nil); err != nil {
		cleanup()
		return nil, err
	}
	if err := ch.QueueBind(queue+".dlq", dlqRoutingKey, dlx, false, nil); err != nil {
		cleanup()
		return nil, err
	}
	return &Consumer{conn: conn, channel: ch, queue: q.Name, dlx: dlx, handler: handler, log: log}, nil
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
			c.process(ctx, d)
		}
	}
}

// process 对单条消息做 panic 隔离：坏消息不得终止整个进程，
// 且 panic 时走 Nack 重试链路，避免未 Ack 消息反复重投形成崩溃循环。
func (c *Consumer) process(ctx context.Context, d amqp.Delivery) {
	defer func() {
		if rec := recover(); rec != nil {
			c.log.ErrorContext(ctx, "event handler panic",
				"queue", c.queue, "panic", rec, "stack", string(debug.Stack()))
			_ = d.Nack(false, false) // 交给 broker 级 DLX
		}
	}()
	if err := c.handle(ctx, d); err != nil {
		c.retryOrDLQ(ctx, d, err)
		return
	}
	if err := d.Ack(false); err != nil {
		c.log.WarnContext(ctx, "ack event failed", "queue", c.queue, "err", err)
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

// retryOrDLQ 只对「已确认成功重新投递」的原消息 Ack；任何投递失败都不 Ack，
// 通过 Nack 让 broker 重投或进死信队列，杜绝「重发失败仍 Ack」导致消息永久丢失。
func (c *Consumer) retryOrDLQ(ctx context.Context, d amqp.Delivery, err error) {
	attempts := deliveryRetryCount(d.Headers)
	if attempts+1 >= MaxRetry {
		c.log.WarnContext(ctx, "event exhausted retries, sending to DLQ",
			"queue", c.queue, "routing_key", d.RoutingKey, "attempts", attempts+1, "err", err)
		// requeue=false + 主队列 DLX 参数 → broker 自动把消息投递到 DLQ。
		if nackErr := d.Nack(false, false); nackErr != nil {
			c.log.ErrorContext(ctx, "nack to DLQ failed", "queue", c.queue, "err", nackErr)
		}
		return
	}

	headers := make(amqp.Table, len(d.Headers)+1)
	for k, v := range d.Headers {
		headers[k] = v
	}
	headers[RetryHeaderKey] = attempts + 1
	// 指数退避 + 抖动，避免重试风暴同步；等待期间可被停机 ctx 取消。
	delay := retryDelayBase << attempts
	select {
	case <-ctx.Done():
		_ = d.Nack(false, true)
		return
	case <-time.After(delay):
	}
	pubCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	publishErr := c.channel.PublishWithContext(pubCtx, "", c.queue, false, false, amqp.Publishing{
		ContentType:  d.ContentType,
		DeliveryMode: amqp.Persistent,
		Headers:      headers,
		Body:         d.Body,
	})
	if publishErr != nil {
		c.log.ErrorContext(ctx, "republish for retry failed, nacking", "queue", c.queue, "err", publishErr)
		_ = d.Nack(false, true)
		return
	}
	if ackErr := d.Ack(false); ackErr != nil {
		c.log.ErrorContext(ctx, "ack after republish failed", "queue", c.queue, "err", ackErr)
	}
}

// queueExchangeDLX 返回死信 exchange 名（与 NewConsumer 中的声明保持一致）。
func (c *Consumer) queueExchangeDLX() string { return c.dlx }

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
