package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"planstar/internal/platform/authn"
	"planstar/internal/platform/authz"
	"planstar/internal/platform/pkg/errcode"
)

// AdminAuthAPI 聚合运营域 JWT 认证与 RBAC 管理端点（ADR-0007）。
// 登录/刷新不经 OpsAuth（否则没有令牌就无法登录），但仍受 CIDR、限流与审计约束；
// 其余端点全部经 OpsAuth.Wrap → 认证 → Casbin 授权。
type AdminAuthAPI struct {
	issuer   *authn.Issuer
	users    authn.UserStore
	enforcer *authz.Enforcer
	ops      *OpsAuth
	logger   *slog.Logger
	limiter  *loginLimiter
}

func NewAdminAuthAPI(issuer *authn.Issuer, users authn.UserStore, enforcer *authz.Enforcer, ops *OpsAuth, logger *slog.Logger) *AdminAuthAPI {
	if logger == nil {
		logger = slog.Default()
	}
	return &AdminAuthAPI{
		issuer: issuer, users: users, enforcer: enforcer, ops: ops,
		logger: logger, limiter: newLoginLimiter(),
	}
}

// Register 把端点挂到 9108 的 mux；不触碰 8098 的 57 条路由清单。
func (api *AdminAuthAPI) Register(mux *http.ServeMux) {
	mux.Handle("POST /admin/auth/login", api.publicGate(http.HandlerFunc(api.login)))
	mux.Handle("POST /admin/auth/refresh", api.publicGate(http.HandlerFunc(api.refresh)))
	mux.Handle("POST /admin/auth/logout", api.ops.Wrap("admin", http.HandlerFunc(api.logout)))
	mux.Handle("GET /admin/rbac/me", api.ops.Wrap("admin", http.HandlerFunc(api.me)))
	mux.Handle("GET /admin/rbac/policies", api.ops.Wrap("admin", http.HandlerFunc(api.listPolicies)))
	mux.Handle("POST /admin/rbac/policies", api.ops.Wrap("admin", http.HandlerFunc(api.addPolicy)))
	mux.Handle("DELETE /admin/rbac/policies", api.ops.Wrap("admin", http.HandlerFunc(api.removePolicy)))
	mux.Handle("GET /admin/rbac/roles", api.ops.Wrap("admin", http.HandlerFunc(api.listRoles)))
	mux.Handle("POST /admin/rbac/users/{name}/roles", api.ops.Wrap("admin", http.HandlerFunc(api.grantRole)))
}

// publicGate 是登录/刷新入口的独立闸门：来源白名单 + 失败限流 + 请求体上限。
func (api *AdminAuthAPI) publicGate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if api.ops != nil && !api.ops.AllowRemote(r.RemoteAddr) {
			writeAdminJSON(w, http.StatusForbidden, map[string]string{
				"code": errcode.Forbidden, "message": "来源地址不允许",
			})
			return
		}
		if !api.limiter.Allow(clientKey(r)) {
			writeAdminJSON(w, http.StatusTooManyRequests, map[string]string{
				"code": errcode.RateLimited, "message": "登录尝试过于频繁，请稍后重试",
			})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
		next.ServeHTTP(w, r)
	})
}

func (api *AdminAuthAPI) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeAdminJSON(w, http.StatusBadRequest, map[string]string{
			"code": errcode.BadRequest, "message": "请求体不是合法 JSON",
		})
		return
	}
	in.Username = strings.TrimSpace(in.Username)
	user, err := api.users.Authenticate(r.Context(), in.Username, in.Password)
	if err != nil {
		api.logger.WarnContext(r.Context(), "admin.auth.login", "audit", true,
			"actor", in.Username, "result", "unauthorized", "remote_addr", r.RemoteAddr)
		writeAdminJSON(w, http.StatusUnauthorized, map[string]string{
			"code": errcode.Unauthorized, "message": "用户名或口令错误",
		})
		return
	}
	access, aid, err := api.issuer.Issue(user.Username, user.UID, user.Roles, authn.Access)
	if err != nil {
		api.internal(w, err)
		return
	}
	refresh, _, err := api.issuer.Issue(user.Username, user.UID, user.Roles, authn.Refresh)
	if err != nil {
		api.internal(w, err)
		return
	}
	api.limiter.Reset(clientKey(r))
	api.logger.InfoContext(r.Context(), "admin.auth.login", "audit", true,
		"actor", user.Username, "result", "ok", "remote_addr", r.RemoteAddr, "roles", strings.Join(user.Roles, ","))
	writeAdminJSON(w, http.StatusOK, map[string]any{
		"tokenType": "Bearer", "accessToken": access, "refreshToken": refresh,
		"expiresIn": int(aid.RemainingTTL().Seconds()),
		"username":  user.Username,
		"roles":     user.Roles,
	})
}

func (api *AdminAuthAPI) refresh(w http.ResponseWriter, r *http.Request) {
	var in struct {
		RefreshToken string `json:"refreshToken"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeAdminJSON(w, http.StatusBadRequest, map[string]string{
			"code": errcode.BadRequest, "message": "请求体不是合法 JSON",
		})
		return
	}
	id, err := api.issuer.Verify(r.Context(), strings.TrimSpace(in.RefreshToken), authn.Refresh)
	if err != nil {
		writeAdminJSON(w, http.StatusUnauthorized, map[string]string{
			"code": errcode.Unauthorized, "message": jwtFailureMessage(err),
		})
		return
	}
	// 账号必须仍存在于目录中，避免已移除的运营账号靠 refresh 续命。
	var user *authn.User
	for _, u := range api.users.All() {
		if u.Username == id.Subject {
			copy := u
			user = &copy
			break
		}
	}
	if user == nil {
		writeAdminJSON(w, http.StatusUnauthorized, map[string]string{
			"code": errcode.Unauthorized, "message": "账号已停用",
		})
		return
	}
	// 轮换：原子消费旧 refresh jti，并发重放只有一个请求能通过（PRODUCTION_AUDIT S3）。
	consumed, err := api.issuer.RevokeOnce(r.Context(), id.JTI, id.RemainingTTL())
	if err != nil {
		api.internal(w, err)
		return
	}
	if !consumed {
		writeAdminJSON(w, http.StatusUnauthorized, map[string]string{
			"code": errcode.Unauthorized, "message": "刷新令牌已使用",
		})
		return
	}
	access, aid, err := api.issuer.Issue(user.Username, user.UID, user.Roles, authn.Access)
	if err != nil {
		api.internal(w, err)
		return
	}
	newRefresh, _, err := api.issuer.Issue(user.Username, user.UID, user.Roles, authn.Refresh)
	if err != nil {
		api.internal(w, err)
		return
	}
	api.logger.InfoContext(r.Context(), "admin.auth.refresh", "audit", true,
		"actor", user.Username, "result", "ok", "remote_addr", r.RemoteAddr)
	writeAdminJSON(w, http.StatusOK, map[string]any{
		"tokenType": "Bearer", "accessToken": access, "refreshToken": newRefresh,
		"expiresIn": int(aid.RemainingTTL().Seconds()), "username": user.Username, "roles": user.Roles,
	})
}

func (api *AdminAuthAPI) logout(w http.ResponseWriter, r *http.Request) {
	id := authn.FromContext(r.Context())
	if id == nil {
		writeAdminJSON(w, http.StatusUnauthorized, map[string]string{
			"code": errcode.Unauthorized, "message": "缺少身份凭据",
		})
		return
	}
	if err := api.issuer.Revoke(r.Context(), id.JTI, id.RemainingTTL()); err != nil {
		api.internal(w, err)
		return
	}
	var in struct {
		RefreshToken string `json:"refreshToken"`
	}
	if r.Body != nil && json.NewDecoder(r.Body).Decode(&in) == nil && strings.TrimSpace(in.RefreshToken) != "" {
		if rid, err := api.issuer.Verify(r.Context(), in.RefreshToken, authn.Refresh); err == nil {
			_ = api.issuer.Revoke(r.Context(), rid.JTI, rid.RemainingTTL())
		}
	}
	api.logger.InfoContext(r.Context(), "admin.auth.logout", "audit", true,
		"actor", id.Subject, "result", "ok", "remote_addr", r.RemoteAddr)
	writeAdminJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (api *AdminAuthAPI) me(w http.ResponseWriter, r *http.Request) {
	id := authn.FromContext(r.Context())
	if id == nil {
		writeAdminJSON(w, http.StatusUnauthorized, map[string]string{
			"code": errcode.Unauthorized, "message": "缺少身份凭据",
		})
		return
	}
	writeAdminJSON(w, http.StatusOK, map[string]any{
		"subject": id.Subject, "uid": id.UID, "roles": id.Roles,
		"subjects": id.Subjects(), "breakGlass": id.ViaBreakGlass,
		"expiresAt": id.ExpiresAt.UnixMilli(),
	})
}

func (api *AdminAuthAPI) listPolicies(w http.ResponseWriter, _ *http.Request) {
	writeAdminJSON(w, http.StatusOK, map[string]any{
		"policies": api.enforcer.Rules(), "grouping": api.enforcer.Grouping(),
	})
}

func (api *AdminAuthAPI) addPolicy(w http.ResponseWriter, r *http.Request) {
	var rule authz.Rule
	if err := decodeRule(w, r, &rule); err != nil {
		writeAdminJSON(w, http.StatusBadRequest, map[string]string{"code": errcode.BadRequest, "message": "请求体不是合法 JSON"})
		return
	}
	// 禁止自我授权：调用方不能给自己新增策略（PRODUCTION_AUDIT S1）。
	if id := authn.FromContext(r.Context()); id != nil && rule.Sub == id.Subject {
		writeAdminJSON(w, http.StatusForbidden, map[string]string{
			"code": errcode.Forbidden, "message": "不能给自己授予策略",
		})
		return
	}
	if err := api.enforcer.AddRule(r.Context(), rule); err != nil {
		writeAdminJSON(w, http.StatusBadRequest, map[string]string{"code": errcode.BadRequest, "message": err.Error()})
		return
	}
	api.auditPolicy(r, "rbac.policy.add", rule.Sub+" "+rule.Obj+" "+rule.Act)
	writeAdminJSON(w, http.StatusOK, map[string]any{"success": true, "policy": rule})
}

func (api *AdminAuthAPI) removePolicy(w http.ResponseWriter, r *http.Request) {
	var rule authz.Rule
	if err := decodeRule(w, r, &rule); err != nil {
		writeAdminJSON(w, http.StatusBadRequest, map[string]string{"code": errcode.BadRequest, "message": err.Error()})
		return
	}
	if err := api.enforcer.RemoveRule(r.Context(), rule); err != nil {
		writeAdminJSON(w, http.StatusBadRequest, map[string]string{"code": errcode.BadRequest, "message": err.Error()})
		return
	}
	api.auditPolicy(r, "rbac.policy.remove", rule.Sub+" "+rule.Obj+" "+rule.Act)
	writeAdminJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (api *AdminAuthAPI) listRoles(w http.ResponseWriter, _ *http.Request) {
	writeAdminJSON(w, http.StatusOK, map[string]any{
		"builtinRoles": []string{
			authn.RoleSuperAdmin, authn.RoleOperator, authn.RoleAuditor, authn.RoleRBACAdmin,
		},
		"grouping": api.enforcer.Grouping(),
		"users":    api.users.All(),
	})
}

func (api *AdminAuthAPI) grantRole(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.PathValue("name"))
	var in struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10)).Decode(&in); err != nil {
		writeAdminJSON(w, http.StatusBadRequest, map[string]string{"code": errcode.BadRequest, "message": "请求体不是合法 JSON"})
		return
	}
	if name == "" || strings.TrimSpace(in.Role) == "" {
		writeAdminJSON(w, http.StatusBadRequest, map[string]string{"code": errcode.BadRequest, "message": "name/role 不能为空"})
		return
	}
	// 禁止自我授权：调用方不能给自己分配角色（PRODUCTION_AUDIT S2）。
	if id := authn.FromContext(r.Context()); id != nil && name == id.Subject {
		writeAdminJSON(w, http.StatusForbidden, map[string]string{
			"code": errcode.Forbidden, "message": "不能给自己分配角色",
		})
		return
	}
	if err := api.enforcer.AddGrouping(r.Context(), authz.Grouping{User: name, Role: strings.TrimSpace(in.Role)}); err != nil {
		writeAdminJSON(w, http.StatusBadRequest, map[string]string{"code": errcode.BadRequest, "message": err.Error()})
		return
	}
	api.auditPolicy(r, "rbac.role.grant", name+" -> "+in.Role)
	writeAdminJSON(w, http.StatusOK, map[string]any{"success": true, "user": name, "role": in.Role})
}

func (api *AdminAuthAPI) auditPolicy(r *http.Request, action, target string) {
	actor := anonymousActor
	if id := authn.FromContext(r.Context()); id != nil {
		actor = id.Subject
	}
	api.logger.InfoContext(r.Context(), action, "audit", true,
		"actor", actor, "target", target, "result", "ok", "remote_addr", r.RemoteAddr)
}

func (api *AdminAuthAPI) internal(w http.ResponseWriter, err error) {
	api.logger.Error("admin auth internal error", "err", err)
	writeAdminJSON(w, http.StatusInternalServerError, map[string]string{
		"code": errcode.Internal, "message": "服务内部错误",
	})
}

func decodeRule(w http.ResponseWriter, r *http.Request, rule *authz.Rule) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10))
	if err := dec.Decode(rule); err != nil {
		return err
	}
	rule.Sub, rule.Obj, rule.Act = strings.TrimSpace(rule.Sub), strings.TrimSpace(rule.Obj), strings.TrimSpace(rule.Act)
	return nil
}

func clientKey(r *http.Request) string {
	return r.RemoteAddr
}

// ---------------------------------------------------------------------------
// 进程内登录失败限流（单实例足够；多实例可替换为 Redis 计数器）。

const (
	maxLoginAttempts = 5
	loginWindow      = time.Minute
)

type loginAttempt struct {
	count int
	reset time.Time
}

type loginLimiter struct {
	mu       sync.Mutex
	attempts map[string]*loginAttempt
}

func newLoginLimiter() *loginLimiter {
	return &loginLimiter{attempts: make(map[string]*loginAttempt)}
}

func (l *loginLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if len(l.attempts) > 1024 {
		for k, a := range l.attempts {
			if now.After(a.reset) {
				delete(l.attempts, k)
			}
		}
	}
	a := l.attempts[key]
	if a == nil || now.After(a.reset) {
		l.attempts[key] = &loginAttempt{count: 1, reset: now.Add(loginWindow)}
		return true
	}
	if a.count >= maxLoginAttempts {
		return false
	}
	a.count++
	return true
}

func (l *loginLimiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, key)
}
