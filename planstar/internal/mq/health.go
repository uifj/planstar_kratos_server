package mq

import (
	"context"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Health 用一次短连接探测 RabbitMQ 可达性。
// 不复用也不修改 Publisher 的长连接，避免探针影响正常发布链路。
func (p *Publisher) Health(ctx context.Context) error {
	if p.addr == "" {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	conn, err := amqp.DialConfig(p.addr, amqp.Config{Dial: amqp.DefaultDial(3 * time.Second)})
	if err != nil {
		return err
	}
	return conn.Close()
}
