package mq

import (
	"context"
	"sync"
	"time"

	"github.com/go-kratos/kratos-layout/internal/biz"
	"github.com/go-kratos/kratos/v3/log"
)

const (
	connectAttempts = 5
	connectBackoff  = 2 * time.Second
)

// Server adapts the RabbitMQ relay + consumers + timeout sweep to
// transport.Server.
//
// App.Run() passes a NON-cancellable context to Start, so this server owns its
// own cancellation: Start blocks until Stop cancels runCtx. Stop waits for
// Start's `done` channel, bounded by kratos.StopTimeout.
type Server struct {
	cfg             Config
	state           *state
	outbox          biz.OutboxRepo
	consumed        biz.ConsumedEventRepo
	orderEvents     biz.OrderEventHandler
	inventoryEvents biz.InventoryEventHandler
	goodsEvents     biz.GoodsEventHandler
	timeout         biz.OrderTimeoutHandler

	runCtx    context.Context
	cancel    context.CancelFunc
	stopOnce  sync.Once
	done      chan struct{}
	startOnce sync.Once
}

func NewServer(
	cfg Config,
	s *state,
	outbox biz.OutboxRepo,
	consumed biz.ConsumedEventRepo,
	orderEvents biz.OrderEventHandler,
	inventoryEvents biz.InventoryEventHandler,
	goodsEvents biz.GoodsEventHandler,
	timeout biz.OrderTimeoutHandler,
) *Server {
	runCtx, cancel := context.WithCancel(context.Background())
	return &Server{
		cfg:             cfg,
		state:           s,
		outbox:          outbox,
		consumed:        consumed,
		orderEvents:     orderEvents,
		inventoryEvents: inventoryEvents,
		goodsEvents:     goodsEvents,
		timeout:         timeout,
		runCtx:          runCtx,
		cancel:          cancel,
		done:            make(chan struct{}),
	}
}

// Start connects (with bounded retries), declares the topology and launches
// the relay, the four consumers and the timeout sweep. If RabbitMQ never
// becomes reachable the server degrades gracefully: HTTP/gRPC keep serving,
// async processing is simply disabled for this process.
func (s *Server) Start(ctx context.Context) error {
	s.startOnce.Do(func() {
		defer close(s.done)
		defer s.state.close()

		if err := s.connectWithRetry(ctx); err != nil {
			log.Warn("mq: rabbitmq unavailable; async processing disabled", "err", err)
			<-s.runCtx.Done()
			return
		}

		relay := newRelay(s.state, s.cfg, s.outbox)
		cons := newConsumers(s.state, s.cfg, s.consumed,
			s.orderEvents, s.inventoryEvents, s.goodsEvents)

		var wg sync.WaitGroup
		wg.Add(3)
		go func() {
			defer wg.Done()
			relay.run(s.runCtx)
		}()
		go func() {
			defer wg.Done()
			cons.run(s.runCtx)
		}()
		go func() {
			defer wg.Done()
			s.timeoutLoop(s.runCtx)
		}()

		<-s.runCtx.Done()
		wg.Wait()
	})
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	s.stopOnce.Do(func() { s.cancel() })
	select {
	case <-s.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Server) connectWithRetry(ctx context.Context) error {
	var err error
	for i := 0; i < connectAttempts; i++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if s.runCtx.Err() != nil {
			return s.runCtx.Err()
		}
		if err = s.state.connect(); err == nil {
			log.Info("mq: connected and topology declared")
			return nil
		}
		if i < connectAttempts-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-s.runCtx.Done():
				return s.runCtx.Err()
			case <-time.After(connectBackoff):
			}
		}
	}
	return err
}

func (s *Server) timeoutLoop(ctx context.Context) {
	t := time.NewTicker(s.cfg.TimeoutLoopInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			before := time.Now().Add(-s.cfg.OrderPayTimeout).Unix()
			if _, err := s.timeout.CancelTimedOut(ctx, before, s.cfg.OutboxBatchSize); err != nil {
				log.Warn("mq: order timeout sweep failed", "err", err)
			}
		}
	}
}

// Compile-time assertion that Server satisfies Kratos' transport contract.
var _ interface {
	Start(context.Context) error
	Stop(context.Context) error
} = (*Server)(nil)
