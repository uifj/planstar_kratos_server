package server

import (
	todov1 "github.com/go-kratos/kratos-layout/api/todo/v1"
	mallv1 "github.com/go-kratos/kratos-layout/api/mall/v1"
	"github.com/go-kratos/kratos-layout/internal/conf"
	"github.com/go-kratos/kratos-layout/internal/service"

	kratosgrpc "github.com/go-kratos/kratos/v3/transport/grpc"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(
	c *conf.Server,
	todo *service.TodoService,
	shop *service.ShopService,
	admin *service.AdminService,
	mw *Middlewares,
) *kratosgrpc.Server {
	var opts = []kratosgrpc.ServerOption{
		kratosgrpc.Middleware(mw.Server...),
		kratosgrpc.StreamMiddleware(mw.Stream...),
	}
	if c.Grpc.Network != "" {
		opts = append(opts, kratosgrpc.Network(c.Grpc.Network))
	}
	if c.Grpc.Addr != "" {
		opts = append(opts, kratosgrpc.Address(c.Grpc.Addr))
	}
	if c.Grpc.Timeout != nil {
		opts = append(opts, kratosgrpc.Timeout(c.Grpc.Timeout.AsDuration()))
	}
	srv := kratosgrpc.NewServer(opts...)

	// Per-operation middleware, matched against canonical RPC operations.
	for _, route := range mw.Routes {
		srv.Use(route.Pattern, route.Middlewares...)
	}

	todov1.RegisterTodoServiceServer(srv, todo)
	mallv1.RegisterShopServiceServer(srv, shop)
	mallv1.RegisterAdminServiceServer(srv, admin)
	return srv
}
