package server

import (
	"log/slog"

	v1 "planstar/admin/api/kratos/admin/v1"
	consolev1 "planstar/admin/api/planstar/admin/v1"
	"planstar/admin/internal/conf"
	"planstar/admin/internal/service"
	"planstar/admin/pkg/auth"
	adminrecovery "planstar/admin/pkg/recovery"
	"planstar/admin/pkg/validate"

	"github.com/go-kratos/kratos/contrib/otel/v3/tracing"
	"github.com/go-kratos/kratos/v3/middleware/logging"
	"github.com/go-kratos/kratos/v3/transport/http"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server, cfg auth.Config, admin *service.AdminService, console *service.ConsoleService, revoked auth.RevocationStore, metrics *Metrics, logger *slog.Logger) *http.Server {
	origins := c.GetHttp().GetCorsAllowOrigins()
	var opts = []http.ServerOption{
		// CORS runs first so a disallowed Origin is rejected before any
		// authentication work; auth.Middleware therefore does not repeat it.
		http.Filter(
			auth.CORS(origins),
			auth.Middleware(cfg, revoked),
		),
		http.Middleware(
			adminrecovery.Recovery(logger),
			logging.Server(logger),
			metrics.Middleware(),
			tracing.Server(),
			validate.Middleware(),
		),
	}
	if c.Http.Network != "" {
		opts = append(opts, http.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, http.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
	}
	srv := http.NewServer(opts...)
	v1.RegisterAdminServiceHTTPServer(srv, admin)
	consolev1.RegisterConsoleServiceHTTPServer(srv, console)
	return srv
}
