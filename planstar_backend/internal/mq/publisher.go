package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"planstar/internal/biz"
)

// Publisher 是 biz.EventPublisher 的 RabbitMQ 实现，带自动重连与 publisher confirm。
// 依据 RabbitMQ 官方 Confirms 文档：写入 socket 不等于投递成功，
// 必须开启 confirm.select 并等待 broker 确认后才能认为事件已落地。
type Publisher struct {
	addr     string
	exchange string

	mu       sync.Mutex
	conn     *amqp.Connection
	ch       *amqp.Channel
	confirms chan amqp.Confirmation
}

func NewPublisher(addr, exchange string) *Publisher {
	return &Publisher{addr: addr, exchange: exchange}
}

// Publish 把领域事件序列化为 JSON，写入 trace 头，并等待 broker confirm。
func (p *Publisher) Publish(ctx context.Context, ev *biz.DomainEvent) error {
	body, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.ensure(); err != nil {
		return err
	}
	headers := make(amqp.Table, 4)
	injectTrace(ctx, headers)
	pubCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := p.ch.PublishWithContext(pubCtx, p.exchange, ev.Type, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Headers:      headers,
		Body:         body,
	}); err != nil {
		return err
	}
	// 等待 confirm：nack / channel 关闭 / 超时都视为发布失败，交由上层记录与补偿。
	select {
	case confirmation, ok := <-p.confirms:
		if !ok {
			return fmt.Errorf("publish confirm channel closed for event %s", ev.Type)
		}
		if !confirmation.Ack {
			return fmt.Errorf("broker nacked event %s (delivery tag %d)", ev.Type, confirmation.DeliveryTag)
		}
		return nil
	case <-pubCtx.Done():
		return fmt.Errorf("publish confirm timeout for event %s: %w", ev.Type, pubCtx.Err())
	}
}

func (p *Publisher) ensure() error {
	if p.conn != nil && !p.conn.IsClosed() && p.ch != nil && p.confirms != nil {
		return nil
	}
	if p.conn != nil {
		_ = p.conn.Close()
	}
	conn, err := amqp.DialConfig(p.addr, amqp.Config{Dial: amqp.DefaultDial(3 * time.Second)})
	if err != nil {
		return err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return err
	}
	if err := ch.ExchangeDeclare(p.exchange, "topic", true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return err
	}
	if err := ch.Confirm(false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return fmt.Errorf("enable publisher confirms: %w", err)
	}
	confirms := make(chan amqp.Confirmation, 1)
	ch.NotifyPublish(confirms)
	p.conn = conn
	p.ch = ch
	p.confirms = confirms
	return nil
}

func (p *Publisher) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ch != nil {
		_ = p.ch.Close()
	}
	if p.conn != nil {
		_ = p.conn.Close()
	}
	p.confirms = nil
}
