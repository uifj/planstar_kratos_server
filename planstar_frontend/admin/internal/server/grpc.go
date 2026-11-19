package server

import (
	"log/slog"

	v1 "planstar/admin/api/kratos/admin/v1"
	"planstar/admin/internal/conf"
	"planstar/admin/internal/service"
	adminrecovery "planstar/admin/pkg/recovery"
	"planstar/admin/pkg/validate"

	"github.com/go-kratos/kratos/contrib/otel/v3/tracing"
	"github.com/go-kratos/kratos/v3/middleware/logging"
	"github.com/go-kratos/kratos/v3/transport/grpc"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(c *conf.Server, admin *service.AdminService, logger *slog.Logger) *grpc.Server {
	var opts = []grpc.ServerOption{
		grpc.Middleware(
			adminrecovery.Recovery(logger),
			logging.Server(logger),
			tracing.Server(),
			validate.Middleware(),
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
	v1.RegisterAdminServiceServer(srv, admin)
	return srv
}
