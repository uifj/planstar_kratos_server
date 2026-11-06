package server

import (
	"net/http"

	todov1 "github.com/go-kratos/kratos-layout/api/todo/v1"
	mallv1 "github.com/go-kratos/kratos-layout/api/mall/v1"
	"github.com/go-kratos/kratos-layout/internal/conf"
	"github.com/go-kratos/kratos-layout/internal/service"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(
	c *conf.Server,
	todo *service.TodoService,
	shop *service.ShopService,
	admin *service.AdminService,
	mw *Middlewares,
) *kratoshttp.Server {
	var opts = []kratoshttp.ServerOption{
		kratoshttp.Middleware(mw.Server...),
	}
	if c.Http.Network != "" {
		opts = append(opts, kratoshttp.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, kratoshttp.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, kratoshttp.Timeout(c.Http.Timeout.AsDuration()))
	}
	srv := kratoshttp.NewServer(opts...)

	// /healthz is registered on the raw router, so it deliberately bypasses
	// the JWT/validate chain. Docker and compose use it as a liveness probe.
	srv.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Per-operation middleware, matched against canonical RPC operations.
	for _, route := range mw.Routes {
		srv.Use(route.Pattern, route.Middlewares...)
	}

	todov1.RegisterTodoServiceHTTPServer(srv, todo)
	mallv1.RegisterShopServiceHTTPServer(srv, shop)
	mallv1.RegisterAdminServiceHTTPServer(srv, admin)
	return srv
}
