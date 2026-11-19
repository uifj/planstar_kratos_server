package auth

import (
	"context"
	"net/http"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/transport"
	httpm "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/google/uuid"
)

// Error reasons reported by this middleware. They mirror the wire-visible names
// in api/<domain>/<version>/error_reason.proto, but are declared here so `pkg`
// does not depend on a domain's generated proto. TestReasonsMatchAPIEnum pins
// them together so they cannot drift.
const (
	reasonUnauthenticated        = "UNAUTHENTICATED"
	reasonPermissionDenied       = "PERMISSION_DENIED"
	reasonPasswordChangeRequired = "PASSWORD_CHANGE_REQUIRED"
	reasonAuthBackend            = "AUTH_BACKEND_UNAVAILABLE"
)

var (
	// noAuthPaths defines the paths that do not require authentication.
	noAuthPaths = map[string]struct{}{
		"/v1/admins/login": {},
	}
	// passwordChangePaths is the only surface a token flagged PwdChange may
	// reach: read the current profile, change the password, or log out.
	passwordChangePaths = map[string]struct{}{
		"/v1/admins/current":  {},
		"/v1/admins/password": {},
		"/v1/admins/logout":   {},
	}
	// ErrUnauthenticated indicates that the request carried no usable credential.
	ErrUnauthenticated = errors.Unauthorized(reasonUnauthenticated, "Token is invalid")
	// ErrPermissionDenied indicates that the caller lacks the required access.
	ErrPermissionDenied = errors.Forbidden(reasonPermissionDenied, "Access denied")
	// ErrPasswordChangeRequired blocks every endpoint except the password-change
	// surface until the admin replaces a temporary password.
	ErrPasswordChangeRequired = errors.Forbidden(reasonPasswordChangeRequired, "Password change required")
	// ErrAuthBackend indicates the revocation store could not be consulted.
	ErrAuthBackend = errors.InternalServer(reasonAuthBackend, "Authentication backend unavailable")
)

// Middleware is an authentication middleware for HTTP servers.
//
// Cross-origin writes are rejected by CORS (see CORS), which runs before this
// filter in the server chain; the origin allow-list is deliberately not
// duplicated here.
func Middleware(cfg Config, store RevocationStore) httpm.FilterFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}
			if _, ok := noAuthPaths[r.URL.Path]; ok {
				next.ServeHTTP(w, r)
				return
			}
			// This filter runs outside the kratos handler chain, so the error
			// encoder has to be invoked explicitly. Writing plain text here
			// instead would leave clients unable to read code / reason.
			cookie, err := r.Cookie(cfg.cookieName())
			if err != nil {
				httpm.DefaultErrorEncoder(w, r, ErrUnauthenticated)
				return
			}
			claims, err := ParseToken(cookie.Value, cfg.Secret)
			if err != nil {
				httpm.DefaultErrorEncoder(w, r, err)
				return
			}
			if store != nil && claims.ID != "" {
				revoked, err := store.Revoked(r.Context(), claims.ID)
				if err != nil {
					httpm.DefaultErrorEncoder(w, r, ErrAuthBackend.WithCause(err))
					return
				}
				if revoked {
					httpm.DefaultErrorEncoder(w, r, ErrUnauthenticated)
					return
				}
			}
			if claims.PwdChange {
				if _, ok := passwordChangePaths[r.URL.Path]; !ok {
					httpm.DefaultErrorEncoder(w, r, ErrPasswordChangeRequired)
					return
				}
			}
			ctx := NewContext(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// SetCookie issues the login cookie with the PwdChange claim embedded.
func SetCookie(ctx context.Context, cfg Config, userID uuid.UUID, access string, mustChange bool, expiresAt time.Time) error {
	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		return errors.New(500, "TRANSPORT_MISSING", "failed to get transport from context")
	}
	ttl := cfg.tokenTTL()
	if expiresAt.IsZero() {
		expiresAt = time.Now().Add(ttl)
	}
	token, err := GenerateToken(userID, access, cfg.Secret, expiresAt, mustChange)
	if err != nil {
		return err
	}
	cookie := &http.Cookie{
		Name:     cfg.cookieName(),
		Value:    token,
		Path:     "/",
		Domain:   cfg.CookieDomain,
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		Secure:   cfg.CookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	tr.ReplyHeader().Add("Set-Cookie", cookie.String())
	return nil
}

// DeleteCookie clears the login cookie in the HTTP response.
func DeleteCookie(ctx context.Context, cfg Config) error {
	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		return errors.New(500, "TRANSPORT_MISSING", "failed to get transport from context")
	}
	cookie := &http.Cookie{
		Name:     cfg.cookieName(),
		Value:    "",
		Path:     "/",
		Domain:   cfg.CookieDomain,
		Expires:  time.Now().AddDate(0, 0, -1),
		MaxAge:   -1,
		Secure:   cfg.CookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	tr.ReplyHeader().Add("Set-Cookie", cookie.String())
	return nil
}
