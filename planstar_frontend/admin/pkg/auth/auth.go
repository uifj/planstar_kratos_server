package auth

import (
	"context"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Role values stored in the admin `access` column.
const (
	RoleViewer   = "viewer"
	RoleOperator = "operator"
	RoleAdmin    = "admin"
)

// Auth user auth.
type Auth struct {
	UserID uuid.UUID `json:"id"`
	Access string    `json:"access"`
	// PwdChange marks a token issued to an admin who must change the password
	// before using the rest of the console.
	PwdChange bool `json:"pwd_change,omitempty"`
	jwt.RegisteredClaims
}

// HasViewerAccess reports whether the caller may read console data.
func (a *Auth) HasViewerAccess() bool {
	switch a.Access {
	case RoleViewer, RoleOperator, RoleAdmin:
		return true
	default:
		return false
	}
}

// HasOperatorAccess reports whether the caller may perform operational writes.
func (a *Auth) HasOperatorAccess() bool {
	switch a.Access {
	case RoleOperator, RoleAdmin:
		return true
	default:
		return false
	}
}

// HasAdminAccess checks if the user has admin access.
func (a *Auth) HasAdminAccess() bool {
	return a.Access == RoleAdmin
}

type authKey struct{}

// NewContext returns a new Context that carries value.
func NewContext(ctx context.Context, auth *Auth) context.Context {
	return context.WithValue(ctx, authKey{}, auth)
}

// FromContext returns the Auth value stored in ctx, if any.
func FromContext(ctx context.Context) (auth *Auth, ok bool) {
	auth, ok = ctx.Value(authKey{}).(*Auth)
	return
}
