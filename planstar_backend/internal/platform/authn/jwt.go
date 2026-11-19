// Package authn 提供运营域 JWT 签发/校验、令牌撤销与账号凭据校验（ADR-0007）。
// 该包属于横切层；biz 层不引用它，保持领域层纯 Go。
package authn

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// 内置角色码。策略文件（authz）以这些码作为 p.sub，不依赖用户目录里的自由文本。
const (
	RoleSuperAdmin = "superadmin"
	RoleOperator   = "operator"
	RoleAuditor    = "auditor"
	RoleRBACAdmin  = "rbac_admin"
)

// TokenType 区分访问令牌与刷新令牌；两者密钥相同但 typ 不可混用。
type TokenType string

const (
	Access  TokenType = "access"
	Refresh TokenType = "refresh"
)

var (
	ErrTokenInvalid   = errors.New("invalid token")
	ErrTokenExpired   = errors.New("token expired")
	ErrTokenRevoked   = errors.New("token revoked")
	ErrWrongTokenType = errors.New("wrong token type")
	ErrBadCredentials = errors.New("bad credentials")
)

// Settings 是 JWT 运行时参数。密钥只从 env 注入，禁止写入 yaml。
type Settings struct {
	Secret     []byte
	Issuer     string
	Audience   string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
	Leeway     time.Duration
}

// DefaultSettings 给出安全默认值（go-wind 的 Claims 过宽，这里刻意最小化）。
func DefaultSettings(secret []byte) Settings {
	return Settings{
		Secret:     secret,
		Issuer:     "planstar.admin",
		Audience:   "planstar.ops",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 7 * 24 * time.Hour,
		Leeway:     30 * time.Second,
	}
}

// Claims 只携带身份与角色码，不携带权限快照（权限变更由 Casbin 即时生效）。
type Claims struct {
	jwt.RegisteredClaims
	Type  TokenType `json:"typ"`
	UID   string    `json:"uid,omitempty"`
	Roles []string  `json:"roles,omitempty"`
}

// Identity 是校验后注入请求 context 的身份，供授权与审计读取。
type Identity struct {
	Subject   string
	UID       string
	Roles     []string
	JTI       string
	Type      TokenType
	ExpiresAt time.Time
	// ViaBreakGlass 标记请求使用的是共享运维令牌而非个人账号，审计必须可区分。
	ViaBreakGlass bool
}

// Subjects 返回参与 Casbin 判定的 subject 列表（用户名 + 角色码）。
func (id *Identity) Subjects() []string {
	if id == nil {
		return nil
	}
	out := make([]string, 0, 1+len(id.Roles))
	if id.Subject != "" {
		out = append(out, id.Subject)
	}
	return append(out, id.Roles...)
}

// Issuer 负责签发、校验与撤销。
type Issuer struct {
	cfg    Settings
	revoke Revocation
}

func NewIssuer(cfg Settings, revoke Revocation) (*Issuer, error) {
	if len(cfg.Secret) < 32 {
		return nil, fmt.Errorf("JWT 密钥至少 32 字节，当前 %d", len(cfg.Secret))
	}
	if strings.TrimSpace(cfg.Issuer) == "" || strings.TrimSpace(cfg.Audience) == "" {
		return nil, fmt.Errorf("JWT issuer/audience 不能为空")
	}
	if cfg.AccessTTL <= 0 || cfg.RefreshTTL <= 0 || cfg.RefreshTTL <= cfg.AccessTTL {
		return nil, fmt.Errorf("JWT TTL 非法：refresh 必须大于 access 且均为正数")
	}
	if cfg.Leeway < 0 || cfg.Leeway > 60*time.Second {
		return nil, fmt.Errorf("JWT clock skew 必须在 0~60s")
	}
	if revoke == nil {
		revoke = NewMemoryRevocation()
	}
	return &Issuer{cfg: cfg, revoke: revoke}, nil
}

func newJTI() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("生成 jti 失败: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// Issue 签发一个带唯一 jti 的令牌，同时返回可直接注入 context 的 Identity。
func (i *Issuer) Issue(subject, uid string, roles []string, typ TokenType) (string, *Identity, error) {
	ttl := i.cfg.AccessTTL
	if typ == Refresh {
		ttl = i.cfg.RefreshTTL
	}
	jti, err := newJTI()
	if err != nil {
		return "", nil, err
	}
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    i.cfg.Issuer,
			Subject:   subject,
			Audience:  jwt.ClaimStrings{i.cfg.Audience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-i.cfg.Leeway)),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			ID:        jti,
		},
		Type:  typ,
		UID:   uid,
		Roles: append([]string(nil), roles...),
	}
	raw, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(i.cfg.Secret)
	if err != nil {
		return "", nil, fmt.Errorf("签发 JWT 失败: %w", err)
	}
	return raw, &Identity{
		Subject: subject, UID: uid, Roles: append([]string(nil), roles...),
		JTI: jti, Type: typ, ExpiresAt: now.Add(ttl),
	}, nil
}

// Verify 完整校验签名、算法白名单、iss/aud/exp/nbf/iat、typ 与 jti 撤销状态。
// 撤销状态查询失败时 fail-closed 返回错误，绝不因 Redis 抖动而放行。
func (i *Issuer) Verify(ctx context.Context, raw string, want TokenType) (*Identity, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, ErrTokenInvalid
	}
	claims := &Claims{}
	tok, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, ErrTokenInvalid
		}
		return i.cfg.Secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(i.cfg.Issuer),
		jwt.WithAudience(i.cfg.Audience),
		jwt.WithLeeway(i.cfg.Leeway),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("%w: %v", ErrTokenInvalid, err)
	}
	if !tok.Valid || claims.ID == "" {
		return nil, ErrTokenInvalid
	}
	if claims.Type != want {
		return nil, fmt.Errorf("%w: got %q want %q", ErrWrongTokenType, claims.Type, want)
	}
	revoked, err := i.revoke.IsRevoked(ctx, claims.ID)
	if err != nil {
		return nil, fmt.Errorf("撤销状态不可用（fail-closed）: %w", err)
	}
	if revoked {
		return nil, ErrTokenRevoked
	}
	return &Identity{
		Subject: claims.Subject, UID: claims.UID,
		Roles: append([]string(nil), claims.Roles...),
		JTI:   claims.ID, Type: claims.Type,
		ExpiresAt: claims.ExpiresAt.Time,
	}, nil
}

// VerifyAccess 是注入 server.OpsAuth 的最小接口适配方法。
func (i *Issuer) VerifyAccess(ctx context.Context, raw string) (*Identity, error) {
	return i.Verify(ctx, raw, Access)
}

// Revoke 撤销单个 jti；ttl 取令牌剩余有效期，避免 Redis 无限增长。
func (i *Issuer) Revoke(ctx context.Context, jti string, ttl time.Duration) error {
	if ttl <= 0 || ttl > i.cfg.RefreshTTL {
		ttl = i.cfg.RefreshTTL
	}
	return i.revoke.Revoke(ctx, jti, ttl)
}

// RevokeOnce 原子消费 jti：仅首次调用返回 true。refresh 轮换必须用它，
// 否则并发请求可在旧令牌被撤销前全部通过校验（PRODUCTION_AUDIT S3）。
func (i *Issuer) RevokeOnce(ctx context.Context, jti string, ttl time.Duration) (bool, error) {
	if ttl <= 0 || ttl > i.cfg.RefreshTTL {
		ttl = i.cfg.RefreshTTL
	}
	return i.revoke.RevokeOnce(ctx, jti, ttl)
}

func (i *Issuer) Revocation() Revocation { return i.revoke }

// RemainingTTL 返回令牌剩余有效期，供登出时设置撤销 key 的过期时间。
func (id *Identity) RemainingTTL() time.Duration {
	if id == nil {
		return 0
	}
	return time.Until(id.ExpiresAt)
}
