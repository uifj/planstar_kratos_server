package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// GenerateToken generates a JWT token for the given user.
func GenerateToken(userID uuid.UUID, access, secret string, expiresAt time.Time, mustChange bool) (string, error) {
	if secret == "" {
		return "", errors.New("auth: jwt secret is empty")
	}
	now := time.Now()
	claims := Auth{
		UserID:    userID,
		Access:    access,
		PwdChange: mustChange,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Issuer:    "kratos",
			Subject:   "user",
			Audience:  []string{"admin"},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// ParseToken parses the JWT token string and returns the Auth claims.
func ParseToken(tokenStr, secret string) (*Auth, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Auth{}, func(token *jwt.Token) (interface{}, error) {
		// Reject any algorithm other than HS256 so a "none"/RS256 token cannot
		// be validated against the shared secret.
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("auth: unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		// Map the jwt driver error to a typed one; returning it raw would make
		// callers see a 500 for a merely expired or malformed token. The cause
		// is kept so logs can still tell expiry from a bad signature.
		return nil, ErrUnauthenticated.WithCause(err)
	}
	if !token.Valid {
		return nil, ErrUnauthenticated
	}
	auth, ok := token.Claims.(*Auth)
	if !ok {
		return nil, ErrUnauthenticated
	}
	return auth, nil
}
