package data

import (
	"context"

	"github.com/go-kratos/kratos-layout/internal/biz"
	"github.com/go-kratos/kratos-layout/internal/data/ent/consumedevent"
)

type consumedEventRepo struct{ data *Data }

func NewConsumedEventRepo(d *Data) biz.ConsumedEventRepo {
	return &consumedEventRepo{data: d}
}

func (r *consumedEventRepo) Exists(ctx context.Context, eventID, consumer string) (bool, error) {
	n, err := r.data.db.ConsumedEvent.Query().
		Where(consumedevent.EventIDEQ(eventID), consumedevent.ConsumerEQ(consumer)).
		Count(ctx)
	return n > 0, err
}

// Mark records that a consumer has processed an event. The unique
// (event_id, consumer) index makes this idempotent: a duplicate insert is
// treated as success rather than an error.
func (r *consumedEventRepo) Mark(ctx context.Context, eventID, consumer string) error {
	_, err := r.data.db.ConsumedEvent.Create().
		SetEventID(eventID).SetConsumer(consumer).Save(ctx)
	if err != nil && isConstraint(err) {
		return nil
	}
	return err
}
