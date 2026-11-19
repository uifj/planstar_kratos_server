package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"planstar/internal/platform/authn"
	"planstar/internal/platform/authz"
	"planstar/internal/platform/conf"
)

type authFixture struct {
	mux      *http.ServeMux
	ops      *OpsAuth
	issuer   *authn.Issuer
	enforcer *authz.Enforcer
	buf      *bytes.Buffer
}

func newAuthFixture(t *testing.T) *authFixture {
	t.Helper()
	ops, buf := newTestOpsAuth(t, conf.Ops{Token: "shared-ops-token"})
	issuer, err := authn.NewIssuer(
		authn.DefaultSettings([]byte("a-very-strong-signing-secret-for-server-tests-2026")),
		authn.NewMemoryRevocation(),
	)
	if err != nil {
		t.Fatalf("NewIssuer: %v", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("alice-pass"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	users, err := authn.NewStaticUserStore([]authn.User{
		{Username: "alice", PasswordHash: string(hash), UID: "u-1", Roles: []string{authn.RoleAuditor}},
	})
	if err != nil {
		t.Fatalf("NewStaticUserStore: %v", err)
	}
	enforcer, err := authz.New(authz.NewMemoryPolicyStore())
	if err != nil {
		t.Fatalf("authz.New: %v", err)
	}
	ops.ConfigureAuthn(issuer, enforcer)
	mux := http.NewServeMux()
	api := NewAdminAuthAPI(issuer, users, enforcer, ops, nil)
	api.Register(mux)
	// 一个需要 auditor 权限的受保护端点，用于验证 RBAC 判定。
	mux.Handle("GET /admin/reports", ops.Wrap("admin", okHandler()))
	return &authFixture{mux: mux, ops: ops, issuer: issuer, enforcer: enforcer, buf: buf}
}

func (f *authFixture) request(t *testing.T, method, path, bearer, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)
	return rec
}

func (f *authFixture) login(t *testing.T) (access, refresh string) {
	t.Helper()
	rec := f.request(t, http.MethodPost, "/admin/auth/login", "", `{"username":"alice","password":"alice-pass"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var out struct {
		AccessToken  string `json:"accessToken"`
		RefreshToken string `json:"refreshToken"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("解析登录响应: %v", err)
	}
	if out.AccessToken == "" || out.RefreshToken == "" {
		t.Fatal("登录响应缺少令牌")
	}
	return out.AccessToken, out.RefreshToken
}

func TestAdminRBACLoginAndAuthorize(t *testing.T) {
	f := newAuthFixture(t)
	access, _ := f.login(t)

	// 所有已认证身份都能查看自身权限。
	if rec := f.request(t, http.MethodGet, "/admin/rbac/me", access, ""); rec.Code != http.StatusOK {
		t.Fatalf("me status = %d body = %s", rec.Code, rec.Body.String())
	}
	// auditor 有举报列表读权限，但没有 RBAC 策略管理权限（401/403 分离）。
	if rec := f.request(t, http.MethodGet, "/admin/reports", access, ""); rec.Code != http.StatusOK {
		t.Errorf("auditor 读举报列表 = %d, want 200", rec.Code)
	}
	if rec := f.request(t, http.MethodGet, "/admin/rbac/policies", access, ""); rec.Code != http.StatusForbidden {
		t.Errorf("auditor 读策略 = %d, want 403", rec.Code)
	}
	// 共享运维令牌映射为显式 superadmin 策略，保证既有运维链路零中断。
	if rec := f.request(t, http.MethodGet, "/admin/rbac/policies", "shared-ops-token", ""); rec.Code != http.StatusOK {
		t.Errorf("ops break-glass 读策略 = %d, want 200", rec.Code)
	}
	// 无凭据必须是 401，而不是 403。
	if rec := f.request(t, http.MethodGet, "/admin/rbac/me", "", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("无凭据 = %d, want 401", rec.Code)
	}
}

func TestAdminRBACRejectsInvalidCredentials(t *testing.T) {
	f := newAuthFixture(t)
	rec := f.request(t, http.MethodPost, "/admin/auth/login", "", `{"username":"alice","password":"wrong"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("错误口令 = %d, want 401", rec.Code)
	}
	if rec := f.request(t, http.MethodGet, "/admin/rbac/me", "not-a-jwt", ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("非法 JWT = %d, want 401", rec.Code)
	}
	// 认证失败不应进入 RBAC 判定阶段。
	if bytes.Contains(f.buf.Bytes(), []byte("rbac denied")) {
		t.Error("认证失败不应产生 rbac denied 审计")
	}
}

func TestAdminRBACLoginRateLimit(t *testing.T) {
	f := newAuthFixture(t)
	for i := 0; i < 5; i++ {
		rec := f.request(t, http.MethodPost, "/admin/auth/login", "", `{"username":"alice","password":"wrong"}`)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("第 %d 次错误登录 = %d, want 401", i+1, rec.Code)
		}
	}
	rec := f.request(t, http.MethodPost, "/admin/auth/login", "", `{"username":"alice","password":"wrong"}`)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("第 6 次登录 = %d, want 429", rec.Code)
	}
}

func TestAdminRBACRefreshRotationAndLogout(t *testing.T) {
	f := newAuthFixture(t)
	access, refresh := f.login(t)

	rec := f.request(t, http.MethodPost, "/admin/auth/refresh", "", `{"refreshToken":"`+refresh+`"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("refresh = %d body = %s", rec.Code, rec.Body.String())
	}
	// 旧 refresh 已被撤销，重放必须失败。
	if rec := f.request(t, http.MethodPost, "/admin/auth/refresh", "", `{"refreshToken":"`+refresh+`"}`); rec.Code != http.StatusUnauthorized {
		t.Errorf("重放旧 refresh = %d, want 401", rec.Code)
	}
	// logout 撤销 access jti。
	if rec := f.request(t, http.MethodPost, "/admin/auth/logout", access, ""); rec.Code != http.StatusOK {
		t.Errorf("logout = %d, want 200", rec.Code)
	}
	if rec := f.request(t, http.MethodGet, "/admin/rbac/me", access, ""); rec.Code != http.StatusUnauthorized {
		t.Errorf("登出后 access 仍可用 = %d, want 401", rec.Code)
	}
}
