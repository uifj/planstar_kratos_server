package authn

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func testIssuer(t *testing.T) *Issuer {
	t.Helper()
	cfg := DefaultSettings([]byte("a-very-strong-signing-secret-for-unit-tests-2026"))
	iss, err := NewIssuer(cfg, NewMemoryRevocation())
	if err != nil {
		t.Fatalf("NewIssuer: %v", err)
	}
	return iss
}

func TestIssueAndVerify(t *testing.T) {
	iss := testIssuer(t)
	raw, id, err := iss.Issue("alice", "u-1", []string{RoleAuditor}, Access)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if id.JTI == "" || id.Subject != "alice" || len(id.Roles) != 1 {
		t.Fatalf("identity = %+v", id)
	}
	got, err := iss.Verify(context.Background(), raw, Access)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if got.Subject != "alice" || got.UID != "u-1" || got.JTI != id.JTI {
		t.Errorf("claims mismatch: %+v", got)
	}
	if len(got.Subjects()) != 2 || got.Subjects()[1] != RoleAuditor {
		t.Errorf("Subjects = %v", got.Subjects())
	}
}

func TestWrongTokenTypeIsRejected(t *testing.T) {
	iss := testIssuer(t)
	raw, _, _ := iss.Issue("alice", "u-1", []string{RoleAuditor}, Refresh)
	if _, err := iss.Verify(context.Background(), raw, Access); !errors.Is(err, ErrWrongTokenType) {
		t.Fatalf("refresh 当 access 用应拒绝，got %v", err)
	}
	if _, err := iss.VerifyAccess(context.Background(), raw); !errors.Is(err, ErrWrongTokenType) {
		t.Fatalf("VerifyAccess 应拒绝 refresh，got %v", err)
	}
}

func TestExpiredAndNoneAlgAreRejected(t *testing.T) {
	iss := testIssuer(t)
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: "planstar.admin", Subject: "alice", Audience: jwt.ClaimStrings{"planstar.ops"},
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
			ID:        "expired-jti",
		},
		Type: Access,
	}
	expired, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(iss.cfg.Secret)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := iss.Verify(context.Background(), expired, Access); !errors.Is(err, ErrTokenExpired) {
		t.Fatalf("过期令牌应返回 ErrTokenExpired，got %v", err)
	}

	noneTok := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	noneRaw, err := noneTok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := iss.Verify(context.Background(), noneRaw, Access); !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("alg=none 必须拒绝，got %v", err)
	}
}

func TestIssuerAudienceMismatchIsRejected(t *testing.T) {
	iss := testIssuer(t)
	raw, _, _ := iss.Issue("alice", "u-1", []string{RoleAuditor}, Access)
	other := testIssuer(t)
	other.cfg.Audience = "another-audience"
	if _, err := other.Verify(context.Background(), raw, Access); !errors.Is(err, ErrTokenInvalid) {
		t.Fatalf("audience 不匹配应拒绝，got %v", err)
	}
}

func TestRevocationIsImmediate(t *testing.T) {
	iss := testIssuer(t)
	raw, id, _ := iss.Issue("alice", "u-1", []string{RoleAuditor}, Access)
	if err := iss.Revoke(context.Background(), id.JTI, time.Minute); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if _, err := iss.Verify(context.Background(), raw, Access); !errors.Is(err, ErrTokenRevoked) {
		t.Fatalf("撤销后应返回 ErrTokenRevoked，got %v", err)
	}
}

func TestRevocationFailureFailsClosed(t *testing.T) {
	cfg := DefaultSettings([]byte("a-very-strong-signing-secret-for-unit-tests-2026"))
	iss, err := NewIssuer(cfg, &failingRevocation{})
	if err != nil {
		t.Fatal(err)
	}
	raw, _, _ := iss.Issue("alice", "u-1", []string{RoleAuditor}, Access)
	if _, err := iss.Verify(context.Background(), raw, Access); err == nil || !strings.Contains(err.Error(), "fail-closed") {
		t.Fatalf("撤销查询失败必须 fail-closed，got %v", err)
	}
}

type failingRevocation struct{}

func (failingRevocation) Revoke(context.Context, string, time.Duration) error { return nil }
func (failingRevocation) RevokeOnce(context.Context, string, time.Duration) (bool, error) {
	return false, errors.New("redis down")
}
func (failingRevocation) IsRevoked(context.Context, string) (bool, error) {
	return false, errors.New("redis down")
}

func TestRevokeOnceIsAtomic(t *testing.T) {
	rev := NewMemoryRevocation()
	ctx := context.Background()
	first, err := rev.RevokeOnce(ctx, "jti-atomic", time.Minute)
	if err != nil || !first {
		t.Fatalf("首次消费应成功: ok=%v err=%v", first, err)
	}
	second, err := rev.RevokeOnce(ctx, "jti-atomic", time.Minute)
	if err != nil || second {
		t.Fatalf("重复消费必须失败: ok=%v err=%v", second, err)
	}
	revoked, err := rev.IsRevoked(ctx, "jti-atomic")
	if err != nil || !revoked {
		t.Fatalf("消费后应处于撤销态: revoked=%v err=%v", revoked, err)
	}
}

func TestNewIssuerRejectsUnsafeSettings(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Settings)
	}{
		{"短密钥", func(s *Settings) { s.Secret = []byte("short") }},
		{"空 issuer", func(s *Settings) { s.Issuer = "" }},
		{"空 audience", func(s *Settings) { s.Audience = "" }},
		{"access TTL 为 0", func(s *Settings) { s.AccessTTL = 0 }},
		{"refresh 不大于 access", func(s *Settings) { s.RefreshTTL = s.AccessTTL }},
		{"leeway 过大", func(s *Settings) { s.Leeway = time.Hour }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := DefaultSettings([]byte("a-very-strong-signing-secret-for-unit-tests-2026"))
			tc.mutate(&cfg)
			if _, err := NewIssuer(cfg, nil); err == nil {
				t.Fatal("不安全配置应被拒绝")
			}
		})
	}
}

func TestStaticUserStore(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("s3cret-pass"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewStaticUserStore([]User{{
		Username: "alice", PasswordHash: string(hash), UID: "u-1", Roles: []string{RoleAuditor},
	}})
	if err != nil {
		t.Fatalf("NewStaticUserStore: %v", err)
	}
	u, err := store.Authenticate(context.Background(), "alice", "s3cret-pass")
	if err != nil || u.Username != "alice" {
		t.Fatalf("正确口令应通过: u=%+v err=%v", u, err)
	}
	if _, err := store.Authenticate(context.Background(), "alice", "wrong"); !errors.Is(err, ErrBadCredentials) {
		t.Errorf("错误口令应返回 ErrBadCredentials，got %v", err)
	}
	if _, err := store.Authenticate(context.Background(), "nobody", "s3cret-pass"); !errors.Is(err, ErrBadCredentials) {
		t.Errorf("不存在的用户应返回 ErrBadCredentials，got %v", err)
	}
}

func TestStaticUserStoreRejectsUnsafeDirectory(t *testing.T) {
	cases := []struct {
		name  string
		users []User
	}{
		{"空用户名", []User{{Username: "", PasswordHash: "x", Roles: []string{"auditor"}}}},
		{"明文口令", []User{{Username: "alice", PasswordHash: "plaintext", Roles: []string{"auditor"}}}},
		{"无角色", []User{{Username: "alice", PasswordHash: "$2a$10$abcdefghijklmnopqrstuuuuuuuuuuuuuuuuuuuuuuuuuuuu"}}},
		{"重复用户", []User{
			{Username: "alice", PasswordHash: "$2a$10$abcdefghijklmnopqrstuuuuuuuuuuuuuuuuuuuuuuuuuuuu", Roles: []string{"auditor"}},
			{Username: "alice", PasswordHash: "$2a$10$abcdefghijklmnopqrstuuuuuuuuuuuuuuuuuuuuuuuuuuuu", Roles: []string{"auditor"}},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewStaticUserStore(tc.users); err == nil {
				t.Fatal("不安全的账号目录应在启动期被拒绝")
			}
		})
	}
}

func TestIdentityContext(t *testing.T) {
	if FromContext(context.Background()) != nil {
		t.Fatal("空 context 不应有身份")
	}
	id := &Identity{Subject: "alice"}
	if got := FromContext(WithIdentity(context.Background(), id)); got != id {
		t.Fatal("WithIdentity/FromContext 不一致")
	}
}
