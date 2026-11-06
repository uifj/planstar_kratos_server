package server

import (
	"log/slog"

	kmetrics "github.com/go-kratos/kratos/contrib/otel/v3/metrics"
	tracing "github.com/go-kratos/kratos/contrib/otel/v3/tracing"
	"github.com/go-kratos/kratos/v3/middleware/logging"
	"github.com/go-kratos/kratos/v3/middleware/metadata"
	"github.com/go-kratos/kratos/v3/middleware/recovery"
	"github.com/go-kratos/kratos/v3/middleware/validate"
	"github.com/go-kratos/kratos/v3/transport/grpc"
	"go.opentelemetry.io/otel"

	v1 "goods/api/goods/v1"
	"goods/internal/conf"
	"goods/internal/service"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(c *conf.Server, goods *service.GoodsService, logger *slog.Logger) *grpc.Server {
	meter := otel.Meter("kratos")
	requests, _ := kmetrics.DefaultRequestsCounter(meter, "server_requests_code_total")
	seconds, _ := kmetrics.DefaultSecondsHistogram(meter, "server_requests_seconds_bucket")
	var opts = []grpc.ServerOption{
		grpc.Middleware(
			recovery.Recovery(),
			metadata.Server(),
			validate.Validator(),
			tracing.Server(),
			kmetrics.Server(
				kmetrics.WithRequests(requests),
				kmetrics.WithSeconds(seconds),
			),
			logging.Server(logger),
		),
	}
	if c.Grpc.Network != "" {
		opts = append(opts, grpc.Network(c.Grpc.Network))
	}
	if c.Grpc.Addr != "" {
		opts = append(opts, grpc.Address(c.Grpc.Addr))
	}
	if c.Grpc.Timeout != nil {
		opts = append(opts, grpc.Timeout(c.Grpc.Timeout.AsDuration()))
	}
	srv := grpc.NewServer(opts...)
	v1.RegisterGoodsServer(srv, goods)
	return srv
}
