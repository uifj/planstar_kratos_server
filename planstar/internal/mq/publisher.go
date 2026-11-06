package mq

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"planstar/internal/biz"
)

// Publisher 是 biz.EventPublisher 的 RabbitMQ 实现，带自动重连。
type Publisher struct {
	addr     string
	exchange string

	mu   sync.Mutex
	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewPublisher(addr, exchange string) *Publisher {
	return &Publisher{addr: addr, exchange: exchange}
}

// Publish 把领域事件序列化为 JSON，并把 trace 上下文写入消息头。
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
	return p.ch.PublishWithContext(pubCtx, p.exchange, ev.Type, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Headers:      headers,
		Body:         body,
	})
}

func (p *Publisher) ensure() error {
	if p.conn != nil && !p.conn.IsClosed() && p.ch != nil {
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
	p.conn = conn
	p.ch = ch
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
}
