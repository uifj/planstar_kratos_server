package data

import (
	"context"
	"errors"
	"time"

	"github.com/go-kratos/kratos-layout/internal/biz"
	"github.com/go-kratos/kratos-layout/internal/conf"

	jwtv5 "github.com/golang-jwt/jwt/v5"
)

const (
	defaultTokenExpire = 30 * 24 * time.Hour
	defaultTokenIssuer = "kratos-mall"
)

// tokenClaims mirrors the JSON keys of server.mallClaims. The two structs are
// deliberately kept separate: data must not import server (that would invert
// the layering), but both must agree on the wire format. Any change to one
// must be mirrored in the other.
type tokenClaims struct {
	ID          string `json:"id"`
	NickName    string `json:"nick_name"`
	AuthorityID int32  `json:"authority_id"`
	jwtv5.RegisteredClaims
}

type tokenManager struct {
	key    []byte
	issuer string
	expire time.Duration
}

// NewTokenManager reuses the single middleware signing key (there is exactly
// one key in this application, which fixes the source project's dual-key drift)
// and reads the expiry/issuer from mall.auth.
func NewTokenManager(m *conf.Mall, keys biz.AuthKeyProvider) biz.TokenManager {
	expire := defaultTokenExpire
	issuer := defaultTokenIssuer
	if m != nil {
		if a := m.GetAuth(); a != nil {
			if a.GetIssuer() != "" {
				issuer = a.GetIssuer()
			}
			if d := a.GetJwtExpire(); d != nil && d.AsDuration() > 0 {
				expire = d.AsDuration()
			}
		}
	}
	return &tokenManager{key: keys.SigningKey(), issuer: issuer, expire: expire}
}

func (m *tokenManager) Generate(_ context.Context, userID, nickName string, authorityID int32) (string, int64, error) {
	if len(m.key) == 0 {
		return "", 0, errors.New("jwt signing key is empty")
	}
	now := time.Now()
	exp := now.Add(m.expire)
	claims := &tokenClaims{
		ID:          userID,
		NickName:    nickName,
		AuthorityID: authorityID,
		RegisteredClaims: jwtv5.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID,
			IssuedAt:  jwtv5.NewNumericDate(now),
			NotBefore: jwtv5.NewNumericDate(now.Add(-time.Minute)),
			ExpiresAt: jwtv5.NewNumericDate(exp),
		},
	}
	signed, err := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims).SignedString(m.key)
	if err != nil {
		return "", 0, err
	}
	return signed, exp.Unix(), nil
}

// Parse performs full claims validation (signature, expiry, not-before). The
// source project's WithoutClaimsValidation allowed expired tokens to be
// refreshed forever; this implementation does not.
func (m *tokenManager) Parse(_ context.Context, token string) (*biz.Claims, error) {
	claims := &tokenClaims{}
	parsed, err := jwtv5.ParseWithClaims(token, claims, func(*jwtv5.Token) (any, error) {
		if len(m.key) == 0 {
			return nil, errors.New("jwt signing key is empty")
		}
		return m.key, nil
	}, jwtv5.WithValidMethods([]string{jwtv5.SigningMethodHS256.Alg()}))
	if err != nil || !parsed.Valid || claims.ID == "" {
		return nil, biz.ErrTokenInvalid
	}
	return &biz.Claims{
		ID:          claims.ID,
		NickName:    claims.NickName,
		AuthorityID: claims.AuthorityID,
	}, nil
}
