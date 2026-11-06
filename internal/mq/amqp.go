package mq

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/go-kratos/kratos-layout/internal/biz"
	amqp "github.com/rabbitmq/amqp091-go"
)

var errNotConnected = errors.New("mq: not connected")

// state owns the single AMQP connection and one dedicated publish channel.
// Each consumer gets its own channel (AMQP best practice); they are created
// in server.go from the shared connection.
type state struct {
	cfg Config

	mu    sync.RWMutex
	conn  *amqp.Connection
	pubCh *amqp.Channel
	pubMu sync.Mutex
}

func NewState(cfg Config) *state { return &state{cfg: cfg} }

func (s *state) connect() error {
	conn, err := amqp.Dial(s.cfg.Addr)
	if err != nil {
		return err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return err
	}
	if err := declareTopology(ch, s.cfg); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return err
	}
	s.mu.Lock()
	s.conn, s.pubCh = conn, ch
	s.mu.Unlock()
	return nil
}

func (s *state) close() {
	s.mu.Lock()
	ch, conn := s.pubCh, s.conn
	s.pubCh, s.conn = nil, nil
	s.mu.Unlock()
	if ch != nil {
		_ = ch.Close()
	}
	if conn != nil {
		_ = conn.Close()
	}
}

func (s *state) consumerChannel() (*amqp.Channel, error) {
	s.mu.RLock()
	conn := s.conn
	s.mu.RUnlock()
	if conn == nil {
		return nil, errNotConnected
	}
	return conn.Channel()
}

// publish serialises writes on the shared publish channel. amqp091 channels
// are not safe for concurrent frame writes, so pubMu guards every publish.
func (s *state) publish(ctx context.Context, exchange, routingKey, messageID string, body []byte, headers amqp.Table) error {
	s.mu.RLock()
	ch := s.pubCh
	s.mu.RUnlock()
	if ch == nil {
		return errNotConnected
	}
	s.pubMu.Lock()
	defer s.pubMu.Unlock()
	return ch.PublishWithContext(ctx, exchange, routingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		MessageId:    messageID,
		Timestamp:    time.Now(),
		Headers:      headers,
		Body:         body,
	})
}

// declareTopology declaration order is fixed: DLX + DLQ first, then the three
// business exchanges, then the durable queues with x-dead-letter-exchange,
// then bindings. Re-declaring with mismatched parameters would close the
// channel, so this function must stay the single source of truth.
func declareTopology(ch *amqp.Channel, c Config) error {
	if err := ch.ExchangeDeclare(c.DLXExchange, "topic", true, false, false, false, nil); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(c.DLQ, true, false, false, false, nil); err != nil {
		return err
	}
	if err := ch.QueueBind(c.DLQ, "#", c.DLXExchange, false, nil); err != nil {
		return err
	}
	for _, ex := range []string{c.OrderExchange, c.InventoryExchange, c.PaymentExchange} {
		if err := ch.ExchangeDeclare(ex, "topic", true, false, false, false, nil); err != nil {
			return err
		}
	}
	args := amqp.Table{"x-dead-letter-exchange": c.DLXExchange}
	for _, q := range []string{
		c.OrderCreatedQueue, c.InventoryResultQueue,
		c.PaymentSuccessQueue, c.OrderPaidGoodsQueue,
	} {
		if _, err := ch.QueueDeclare(q, true, false, false, false, args); err != nil {
			return err
		}
	}
	bindings := []struct{ queue, key, exchange string }{
		{c.OrderCreatedQueue, biz.EventOrderCreated, c.OrderExchange},
		{c.InventoryResultQueue, biz.EventInventoryLockOK, c.InventoryExchange},
		{c.InventoryResultQueue, biz.EventInventoryLockFailed, c.InventoryExchange},
		{c.PaymentSuccessQueue, biz.EventPaymentSuccess, c.PaymentExchange},
		{c.OrderPaidGoodsQueue, biz.EventOrderPaid, c.PaymentExchange},
	}
	for _, b := range bindings {
		if err := ch.QueueBind(b.queue, b.key, b.exchange, false, nil); err != nil {
			return err
		}
	}
	return nil
}
