package mq

import (
	"context"
	"time"

	"github.com/go-kratos/kratos-layout/internal/biz"
	"github.com/go-kratos/kratos/v3/log"
)

// relay implements the transactional-outbox publisher loop. Business rows
// and the outbox row are committed together (in data layer); this loop only
// ever publishes rows that already exist, so no event can be lost.
type relay struct {
	state  *state
	cfg    Config
	outbox biz.OutboxRepo
}

func newRelay(s *state, cfg Config, outbox biz.OutboxRepo) *relay {
	return &relay{state: s, cfg: cfg, outbox: outbox}
}

func (r *relay) run(ctx context.Context) {
	t := time.NewTicker(r.cfg.OutboxScanInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			r.publishPending(ctx)
		}
	}
}

func (r *relay) publishPending(ctx context.Context) {
	rows, err := r.outbox.ListPending(ctx, r.cfg.OutboxBatchSize)
	if err != nil {
		log.Warn("outbox scan failed", "err", err)
		return
	}
	for _, e := range rows {
		ex := exchangeForEventType(e.EventType)
		if ex == "" {
			_ = r.outbox.MarkFailed(ctx, e.ID)
			continue
		}
		err := r.state.publish(ctx, r.cfg.exchange(ex), e.EventType,
			e.EventID, []byte(e.Payload), nil)
		if err != nil {
			_ = r.outbox.IncrRetry(ctx, e.ID)
			if e.RetryCount+1 >= int32(r.cfg.MaxRetry) {
				_ = r.outbox.MarkFailed(ctx, e.ID)
			}
			continue
		}
		_ = r.outbox.MarkPublished(ctx, e.ID)
	}
}
