package server

import (
	v1 "github.com/go-kratos/kratos-layout/api/todo/v1"
	"github.com/go-kratos/kratos-layout/internal/conf"
	"github.com/go-kratos/kratos-layout/internal/service"

	"github.com/go-kratos/kratos/v3/transport/grpc"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(c *conf.Server, todo *service.TodoService, mw *Middlewares) *grpc.Server {
	var opts = []grpc.ServerOption{
		grpc.Middleware(mw.Server...),
		grpc.StreamMiddleware(mw.Stream...),
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
	// Per-operation middleware, matched against canonical RPC operations.
	for _, route := range mw.Routes {
		srv.Use(route.Pattern, route.Middlewares...)
	}
	v1.RegisterTodoServiceServer(srv, todo)
	return srv
}
