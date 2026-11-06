package data

import (
	"context"

	"github.com/go-kratos/kratos-layout/internal/biz"
	"github.com/go-kratos/kratos-layout/internal/data/ent"
	"github.com/go-kratos/kratos-layout/internal/data/ent/ordereventoutbox"
)

func outboxToBiz(po *ent.OrderEventOutbox) *biz.OutboxEvent {
	if po == nil {
		return nil
	}
	return &biz.OutboxEvent{
		ID: idStr(po.ID), EventID: po.EventID, EventType: po.EventType,
		OrderSn: po.OrderSn, Payload: po.Payload, Status: po.Status,
		RetryCount: po.RetryCount,
	}
}

type outboxRepo struct{ data *Data }

func NewOutboxRepo(d *Data) biz.OutboxRepo { return &outboxRepo{data: d} }

// ListPending returns the oldest pending rows first. UUIDv7 ids are
// time-sortable, so ordering by id gives a stable FIFO without an extra
// sequence column.
func (r *outboxRepo) ListPending(ctx context.Context, limit int) ([]*biz.OutboxEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	pos, err := r.data.db.OrderEventOutbox.Query().
		Where(ordereventoutbox.StatusEQ(biz.OutboxPending)).
		Order(ordereventoutbox.ByID()).Limit(limit).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*biz.OutboxEvent, 0, len(pos))
	for _, po := range pos {
		out = append(out, outboxToBiz(po))
	}
	return out, nil
}

func (r *outboxRepo) MarkPublished(ctx context.Context, id string) error {
	_, err := r.data.db.OrderEventOutbox.UpdateOneID(parseID(id)).
		SetStatus(biz.OutboxPublished).Save(ctx)
	return err
}

func (r *outboxRepo) MarkFailed(ctx context.Context, id string) error {
	_, err := r.data.db.OrderEventOutbox.UpdateOneID(parseID(id)).
		SetStatus(biz.OutboxFailed).Save(ctx)
	return err
}

func (r *outboxRepo) IncrRetry(ctx context.Context, id string) error {
	_, err := r.data.db.OrderEventOutbox.UpdateOneID(parseID(id)).
		AddRetryCount(1).Save(ctx)
	return err
}

func (r *outboxRepo) Create(ctx context.Context, e *biz.OutboxEvent) (*biz.OutboxEvent, error) {
	po, err := r.data.db.OrderEventOutbox.Create().
		SetID(parseID(e.ID)).SetEventID(e.EventID).
		SetEventType(e.EventType).SetOrderSn(e.OrderSn).
		SetPayload(e.Payload).SetStatus(e.Status).
		SetRetryCount(e.RetryCount).Save(ctx)
	if err != nil {
		return nil, err
	}
	return outboxToBiz(po), nil
}
