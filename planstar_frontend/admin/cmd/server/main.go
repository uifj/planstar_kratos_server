package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"strings"
	"time"

	"planstar/admin/internal/conf"
	"planstar/admin/internal/server"
	"planstar/admin/pkg/auth"

	"github.com/go-kratos/kratos/contrib/otel/v3/tracing"
	"github.com/go-kratos/kratos/v3"
	"github.com/go-kratos/kratos/v3/config"
	"github.com/go-kratos/kratos/v3/config/env"
	"github.com/go-kratos/kratos/v3/config/file"
	"github.com/go-kratos/kratos/v3/log"
	"github.com/go-kratos/kratos/v3/transport/grpc"
	"github.com/go-kratos/kratos/v3/transport/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	_ "go.uber.org/automaxprocs"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Name is the name of the compiled software.
	Name string
	// Version is the version of the compiled software.
	Version string
	// flagconf is the config flag.
	flagconf string

	id, _ = os.Hostname()
)

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs", "config path, eg: -conf config.yaml")
}

func newAuthConfig(c *conf.Auth) auth.Config {
	return auth.Config{
		Secret:       c.GetJwtSecret(),
		CookieName:   c.GetCookieName(),
		CookieDomain: c.GetCookieDomain(),
		CookieSecure: c.GetCookieSecure(),
		TokenTTL:     c.GetTokenTtl().AsDuration(),
	}
}

func newApp(logger *slog.Logger, gs *grpc.Server, hs *http.Server, ops *server.OpsServer) *kratos.App {
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(
			gs,
			hs,
			ops,
		),
	)
}

// initTracing wires an OTLP trace exporter only when an endpoint is
// configured. Without it tracing stays a no-op, so local development needs no
// collector. The exporter connects lazily, so startup is never blocked on it.
func initTracing(ctx context.Context) (func(), error) {
	endpoint := strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	if endpoint == "" {
		return func() {}, nil
	}
	opts := []otlptracegrpc.Option{otlptracegrpc.WithTimeout(5 * time.Second)}
	if strings.Contains(endpoint, "://") {
		opts = append(opts, otlptracegrpc.WithEndpointURL(endpoint))
	} else {
		opts = append(opts, otlptracegrpc.WithEndpoint(endpoint), otlptracegrpc.WithInsecure())
	}
	exporter, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		return func() {}, err
	}
	provider := sdktrace.NewTracerProvider(sdktrace.WithBatcher(exporter))
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = provider.Shutdown(shutdownCtx)
	}, nil
}

func main() {
	flag.Parse()
	logger := log.NewLogger(
		slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelInfo,
		}),
		log.WithExtractor(tracing.TraceAttrs),
	).With(
		slog.String("service.id", id),
		slog.String("service.name", Name),
		slog.String("service.version", Version),
	)
	log.SetDefault(logger)
	c := config.New(
		config.WithSource(
			file.NewSource(flagconf),
			env.NewSource("KRATOS"),
			env.NewSource("ADMIN"),
		),
		config.WithDecoder(conf.DecodeConfig),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}
	if err := conf.Validate(&bc); err != nil {
		panic(err)
	}

	shutdownTracing, err := initTracing(context.Background())
	if err != nil {
		// A misconfigured collector must not keep the service from starting.
		log.Warn("tracing disabled", "error", err)
	}
	defer shutdownTracing()

	app, cleanup, err := wireApp(bc.Server, bc.Data, bc.Auth, bc.Planstar, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	// start and wait for stop signal
	if err := app.Run(); err != nil {
		panic(err)
	}
}
