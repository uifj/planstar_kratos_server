package biz

import "context"

// Authority levels stored in the users.role column and carried in JWT claims.
const (
	AuthorityUser  int32 = 1
	AuthorityAdmin int32 = 2
)

// Claims is the pure-Go identity of an authenticated caller. The JWT-shaped
// struct lives in internal/server; the transport middleware translates it into
// this value before calling into service, so biz and service never import a
// jwt package or a transport package.
type Claims struct {
	ID          string
	NickName    string
	AuthorityID int32
}

type claimsContextKey struct{}

// WithClaims attaches the authenticated identity to a request context. Only
// the server middleware calls this.
func WithClaims(ctx context.Context, c *Claims) context.Context {
	if c == nil {
		return ctx
	}
	return context.WithValue(ctx, claimsContextKey{}, c)
}

// ClaimsFromContext returns the authenticated identity, if any.
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(claimsContextKey{}).(*Claims)
	return c, ok && c != nil
}

// UserIDFromContext returns the authenticated user id. Handlers that require
// an identity use this instead of trusting a client-supplied uid.
func UserIDFromContext(ctx context.Context) (string, error) {
	c, ok := ClaimsFromContext(ctx)
	if !ok || c.ID == "" {
		return "", ErrTokenInvalid
	}
	return c.ID, nil
}
