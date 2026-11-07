// Package recovery provides a panic recovery middleware that never writes raw
// request payloads to the log.
package recovery

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/middleware"
)

// ErrUnknownRequest is returned to the caller when a handler panics.
var ErrUnknownRequest = errors.InternalServer("UNKNOWN", "unknown request error")

// Recovery is a server middleware that recovers from any panics. It mirrors
// the Kratos recovery middleware but logs a redacted request description
// instead of the raw request object.
func Recovery(logger *slog.Logger) middleware.Middleware {
	if logger == nil {
		logger = slog.Default()
	}
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (reply any, err error) {
			defer func() {
				if rerr := recover(); rerr != nil {
					buf := make([]byte, 64<<10) //nolint:mnd
					n := runtime.Stack(buf, false)
					buf = buf[:n]
					logger.ErrorContext(ctx, "panic recovered",
						slog.Any("panic", rerr),
						slog.String("request", Describe(req)),
						slog.String("stack", string(buf)),
					)
					err = ErrUnknownRequest
				}
			}()
			return handler(ctx, req)
		}
	}
}

// Describe renders req for panic logs without exposing its payload. Requests
// that implement Redact control their own redaction; every other request is
// reduced to its type name so unknown secrets never reach the log.
func Describe(req any) string {
	if r, ok := req.(interface{ Redact() string }); ok {
		return r.Redact()
	}
	return fmt.Sprintf("%T", req)
}
