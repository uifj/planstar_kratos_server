package server

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/go-kratos/kratos-layout/internal/biz"
	"github.com/go-kratos/kratos-layout/internal/conf"

	kratosjwt "github.com/go-kratos/kratos/contrib/middleware/jwt/v3"
	otelmetrics "github.com/go-kratos/kratos/contrib/otel/v3/metrics"
	oteltracing "github.com/go-kratos/kratos/contrib/otel/v3/tracing"
	"github.com/go-kratos/kratos/v3/log"
	krmetadata "github.com/go-kratos/kratos/v3/metadata"
	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/middleware/circuitbreaker"
	"github.com/go-kratos/kratos/v3/middleware/logging"
	mwmetadata "github.com/go-kratos/kratos/v3/middleware/metadata"
	"github.com/go-kratos/kratos/v3/middleware/ratelimit"
	"github.com/go-kratos/kratos/v3/middleware/recovery"
	"github.com/go-kratos/kratos/v3/middleware/selector"
	"github.com/go-kratos/kratos/v3/middleware/validate"
	"github.com/go-kratos/kratos/v3/transport"
	jwtv5 "github.com/golang-jwt/jwt/v5"
	"go.einride.tech/aip/fieldbehavior"
	"go.opentelemetry.io/otel"
	"google.golang.org/protobuf/proto"
)

// This file is the composition root for every middleware documented at
// https://go-kratos.dev/zh-cn/docs/component/middleware/overview/ :
//
//	core:     circuitbreaker, logging, metadata, ratelimit, recovery,
//	          selector, validate
//	contrib:  jwt (auth), otel tracing, otel metrics
//
// Layering, per AGENTS.md:
//   - biz declares the policy ports (RateLimiter, CircuitBreakerFactory,
//     AuthKeyProvider) as plain Go interfaces.
//   - data implements those ports; this file never reads config values or
//     credentials directly.
//   - server only composes middleware; it contains no business rules and no
//     translation.
//   - cmd/server wires the ports and the composition root together via Wire.
//
// All chains are built from configuration, so a middleware can be toggled
// without recompiling or editing this file.

const defaultServiceName = "kratos-layout"

// RouteMiddleware binds extra middleware to canonical RPC operations via
// http.Server.Use / grpc.Server.Use. Patterns are operations such as
// "/todo.v1.TodoService/*", not HTTP paths.
type RouteMiddleware struct {
	Pattern     string
	Middlewares []middleware.Middleware
}

// Middlewares holds the composed chains.
//   - Server: unary HTTP and gRPC inbound calls.
//   - Stream: gRPC streaming calls (separate lifetime, no per-request
//     admission control).
//   - Client: outbound calls, ready for grpc.WithMiddleware(...) /
//     http.WithMiddleware(...).
//   - Routes: per-operation extras installed with Use(pattern, ...).
type Middlewares struct {
	Server []middleware.Middleware
	Stream []middleware.Middleware
	Client []middleware.Middleware
	Routes []RouteMiddleware
}

// ClientMiddleware composes the outbound chain with middleware.Chain.
// Install it on a future grpc.NewClient / http.NewClient as
// grpc.WithMiddleware(mw.ClientMiddleware()).
func (m *Middlewares) ClientMiddleware() middleware.Middleware {
	return middleware.Chain(m.Client...)
}

// NewMiddlewares assembles all documented middleware. The order follows the
// overview page: metadata and tracing first so the rest of the chain has
// their context; logging outside recovery so recovered panics appear in the
// completion record; rate limiting and auth close to the handler; validation
// immediately before it.
func NewMiddlewares(
	c *conf.Middleware,
	logger *slog.Logger,
	limiter biz.RateLimiter,
	breakers biz.CircuitBreakerFactory,
	keys biz.AuthKeyProvider,
) (*Middlewares, error) {
	if c == nil {
		c = &conf.Middleware{}
	}
	if logger == nil {
		logger = slog.Default()
	}

	mdOpts := metadataOptions(c)
	tracerName := orDefault(c.GetTracerName(), defaultServiceName)
	tracingOpts := []oteltracing.Option{oteltracing.WithTracerName(tracerName)}

	serverMW := make([]middleware.Middleware, 0, 8)
	if c.GetMetadata() {
		serverMW = append(serverMW, mwmetadata.Server(mdOpts...))
	}
	if c.GetTracing() {
		serverMW = append(serverMW, oteltracing.Server(tracingOpts...))
	}
	if c.GetMetrics() {
		opts, err := metricOptions(orDefault(c.GetMeterName(), defaultServiceName), true)
		if err != nil {
			return nil, err
		}
		serverMW = append(serverMW, otelmetrics.Server(opts...))
	}
	if c.GetLogging() {
		serverMW = append(serverMW, logging.Server(logger))
	}
	if c.GetRecovery() {
		serverMW = append(serverMW, recovery.Recovery(recovery.WithLogger(logger)))
	}
	if c.GetRatelimit() {
		serverMW = append(serverMW, ratelimit.Server(ratelimit.WithLimiter(limiterAdapter{limiter})))
	}
	if c.GetJwt() {
		serverMW = append(serverMW, authMiddleware(c, keys))
	}
	if c.GetValidate() {
		serverMW = append(serverMW, validate.Validator(validateRequests))
	}

	// Streaming RPCs are long-lived. Keep context, observability, auth and
	// validation, but skip per-request rate limiting and metrics.
	streamMW := make([]middleware.Middleware, 0, 6)
	if c.GetMetadata() {
		streamMW = append(streamMW, mwmetadata.Server(mdOpts...))
	}
	if c.GetTracing() {
		streamMW = append(streamMW, oteltracing.Server(tracingOpts...))
	}
	if c.GetLogging() {
		streamMW = append(streamMW, logging.Server(logger))
	}
	if c.GetRecovery() {
		streamMW = append(streamMW, recovery.Recovery(recovery.WithLogger(logger)))
	}
	if c.GetJwt() {
		streamMW = append(streamMW, authMiddleware(c, keys))
	}
	if c.GetValidate() {
		streamMW = append(streamMW, validate.Validator(validateRequests))
	}

	clientMW := make([]middleware.Middleware, 0, 6)
	if c.GetMetadata() {
		clientMW = append(clientMW, mwmetadata.Client(mdOpts...))
	}
	if c.GetTracing() {
		clientMW = append(clientMW, oteltracing.Client(tracingOpts...))
	}
	if c.GetMetrics() {
		opts, err := metricOptions(orDefault(c.GetMeterName(), defaultServiceName), false)
		if err != nil {
			return nil, err
		}
		clientMW = append(clientMW, otelmetrics.Client(opts...))
	}
	if c.GetLogging() {
		clientMW = append(clientMW, logging.Client(logger))
	}
	if c.GetCircuitBreaker() {
		clientMW = append(clientMW, circuitbreaker.Client(breakerFactoryOption(breakers)))
	}
	if c.GetJwtClient() {
		clientMW = append(clientMW, kratosjwt.Client(jwtKeyFunc(keys),
			kratosjwt.WithSigningMethod(jwtv5.SigningMethodHS256),
			kratosjwt.WithClaims(func() jwtv5.Claims { return &jwtv5.RegisteredClaims{} }),
		))
	}

	// Use(pattern, ...) applies middleware only to selected canonical
	// operations. selector.Server(...) does the inverse: it applies JWT to
	// everything except the configured public allowlist.
	routes := make([]RouteMiddleware, 0, 1)
	if c.GetLogging() {
		routes = append(routes, RouteMiddleware{
			Pattern:     "/todo.v1.TodoService/*",
			Middlewares: []middleware.Middleware{audit(logger)},
		})
	}

	return &Middlewares{
		Server: serverMW,
		Stream: streamMW,
		Client: clientMW,
		Routes: routes,
	}, nil
}

// metadataOptions configures constants and propagated prefixes.
func metadataOptions(c *conf.Middleware) []mwmetadata.Option {
	constants := krmetadata.Metadata{}
	constants.Set("x-md-service", defaultServiceName)
	opts := []mwmetadata.Option{mwmetadata.WithConstants(constants)}
	if prefixes := c.GetPropagatedPrefix(); len(prefixes) > 0 {
		opts = append(opts, mwmetadata.WithPropagatedPrefix(prefixes...))
	}
	return opts
}

// metricOptions creates the OTel instruments recommended by the metrics doc.
// The application owns the meter provider; Kratos only records into the
// instruments it is given.
func metricOptions(meterName string, server bool) ([]otelmetrics.Option, error) {
	meter := otel.Meter(meterName)
	var requestsName, secondsName string
	if server {
		requestsName = otelmetrics.DefaultServerRequestsCounterName
		secondsName = otelmetrics.DefaultServerSecondsHistogramName
	} else {
		requestsName = otelmetrics.DefaultClientRequestsCounterName
		secondsName = otelmetrics.DefaultClientSecondsHistogramName
	}
	requests, err := otelmetrics.DefaultRequestsCounter(meter, requestsName)
	if err != nil {
		return nil, err
	}
	seconds, err := otelmetrics.DefaultSecondsHistogram(meter, secondsName)
	if err != nil {
		return nil, err
	}
	return []otelmetrics.Option{
		otelmetrics.WithRequests(requests),
		otelmetrics.WithSeconds(seconds),
	}, nil
}

// authMiddleware applies JWT only to operations that are not on the public
// allowlist. selector inspects canonical RPC operations, not HTTP paths.
func authMiddleware(c *conf.Middleware, keys biz.AuthKeyProvider) middleware.Middleware {
	jwtMW := kratosjwt.Server(
		jwtKeyFunc(keys),
		kratosjwt.WithSigningMethod(jwtv5.SigningMethodHS256),
		kratosjwt.WithClaims(func() jwtv5.Claims { return &jwtv5.RegisteredClaims{} }),
	)
	public := make(map[string]struct{}, len(c.GetPublicOperations()))
	for _, op := range c.GetPublicOperations() {
		public[op] = struct{}{}
	}
	return selector.Server(jwtMW).Match(func(_ context.Context, operation string) bool {
		_, isPublic := public[operation]
		return !isPublic
	}).Build()
}

func jwtKeyFunc(keys biz.AuthKeyProvider) jwtv5.Keyfunc {
	return func(*jwtv5.Token) (any, error) {
		key := keys.SigningKey()
		if len(key) == 0 {
			return nil, errors.New("jwt signing key is empty")
		}
		return key, nil
	}
}

// validateRequests is the AIP field-behavior validator already used by the
// template; it runs at the transport boundary before the service handler.
func validateRequests(v any) error {
	if msg, ok := v.(proto.Message); ok {
		return fieldbehavior.ValidateRequiredFields(msg)
	}
	return nil
}

// audit is the custom middleware from the overview page's "编写中间件"
// section. It adds request-scoped log attributes via log.ContextWithAttrs and
// calls next exactly once.
func audit(logger *slog.Logger) middleware.Middleware {
	return func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			var operation string
			if tr, ok := transport.FromServerContext(ctx); ok {
				operation = tr.Operation()
			}
			ctx = log.ContextWithAttrs(ctx, slog.String("operation", operation))
			start := time.Now()
			reply, err := next(ctx, req)
			logger.DebugContext(ctx, "route audit completed",
				"elapsed", time.Since(start), "error", err)
			return reply, err
		}
	}
}

// --- adapters from biz ports to the Kratos middleware contracts ---------
// These are pure wiring: no policy decisions live here.

type limiterAdapter struct{ inner biz.RateLimiter }

func (a limiterAdapter) Allow() (ratelimit.DoneFunc, error) {
	done, err := a.inner.Allow()
	if err != nil {
		return nil, err
	}
	if done == nil {
		return nil, biz.ErrRateLimitExceeded
	}
	return func(info ratelimit.DoneInfo) { done(info.Err) }, nil
}

type breakerAdapter struct{ inner biz.CircuitBreaker }

func (a breakerAdapter) Allow() error { return a.inner.Allow() }
func (a breakerAdapter) MarkSuccess() { a.inner.MarkSuccess() }
func (a breakerAdapter) MarkFailed()  { a.inner.MarkFailed() }

func breakerFactoryOption(f biz.CircuitBreakerFactory) circuitbreaker.Option {
	return circuitbreaker.WithBreakerFactory(func() circuitbreaker.CircuitBreaker {
		return breakerAdapter{inner: f.New()}
	})
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
