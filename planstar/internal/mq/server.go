package mq

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"planstar/internal/conf"
)

// Server 把 RabbitMQ 消费者接入 kratos.App 的生命周期（实现 transport.Server）。
type Server struct {
	conf    conf.MQ
	handler Handler
	log     *slog.Logger

	mu       sync.Mutex
	consumer *Consumer
	cancel   context.CancelFunc
	done     chan struct{}
}

func NewServer(c conf.MQ, handler Handler, log *slog.Logger) *Server {
	return &Server{conf: c, handler: handler, log: log, done: make(chan struct{})}
}

func (s *Server) Start(ctx context.Context) error {
	if s.conf.Addr == "" {
		s.log.Warn("rabbitmq disabled: mq.addr is empty")
		return nil
	}
	runCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	go s.loop(runCtx)
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	if s.cancel != nil {
		s.cancel()
	}
	select {
	case <-s.done:
	case <-ctx.Done():
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.consumer != nil {
		s.consumer.Close()
		s.consumer = nil
	}
	return nil
}

// loop 支持断线重连：连接失败时间隔 5s 重试，直到进程退出。
func (s *Server) loop(ctx context.Context) {
	defer close(s.done)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		c, err := NewConsumer(s.conf.Addr, s.conf.Exchange, s.conf.Queue,
			[]string{"user.registered", "order.created"}, s.handler)
		if err != nil {
			s.log.WarnContext(ctx, "rabbitmq consumer connect failed, retrying", "err", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
				continue
			}
		}
		s.mu.Lock()
		s.consumer = c
		s.mu.Unlock()
		s.log.InfoContext(ctx, "rabbitmq consumer started", "queue", s.conf.Queue)

		if err := c.Run(ctx); err != nil {
			s.log.WarnContext(ctx, "rabbitmq consumer stopped", "err", err)
		}
		c.Close()
		s.mu.Lock()
		s.consumer = nil
		s.mu.Unlock()

		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}
	}
}
