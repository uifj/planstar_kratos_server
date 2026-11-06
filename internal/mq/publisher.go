package mq

import (
	"context"
	"encoding/json"

	"github.com/go-kratos/kratos-layout/internal/biz"
)

// publisher implements biz.EventPublisher. It is injected into
// InventoryUsecase at Wire time, before the AMQP connection exists, so it
// reads the publish channel lazily on every call.
type publisher struct {
	state *state
	cfg   Config
}

func NewPublisher(s *state, cfg Config) biz.EventPublisher {
	return &publisher{state: s, cfg: cfg}
}

func (p *publisher) Publish(ctx context.Context, exchange, routingKey string, event *biz.DomainEvent) error {
	if event == nil {
		return nil
	}
	body, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return p.state.publish(ctx, p.cfg.exchange(exchange), routingKey, event.ID, body, nil)
}
