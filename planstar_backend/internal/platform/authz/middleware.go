package authz

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"planstar/internal/platform/authn"
	"planstar/internal/platform/pkg/errcode"
)

// Authorizer 是 server 层依赖的最小授权接口（ADR-0007 D5），由 Enforcer 实现。
type Authorizer interface {
	Authorize(ctx context.Context, subjects []string, obj, act string) (bool, error)
}

// Middleware 是认证之后的第二阶段：从 context 取身份，按 method+path 做 RBAC 判定。
// 认证失败 401、越权 403 分离；判定出错一律拒绝（fail-closed）。
type Middleware struct {
	authorizer Authorizer
	log        *slog.Logger
}

func NewMiddleware(a Authorizer, log *slog.Logger) *Middleware {
	if log == nil {
		log = slog.Default()
	}
	return &Middleware{authorizer: a, log: log}
}

// Decision 供 server.OpsAuth 直接调用，避免把 HTTP 包装逻辑复制两份。
func (m *Middleware) Decision(ctx context.Context, id *authn.Identity, obj, act string) (allowed bool, reason string, err error) {
	if id == nil {
		return false, "缺少身份凭据", nil
	}
	if m.authorizer == nil {
		// 未装配授权器时拒绝，绝不退化成全放行。
		return false, "RBAC 授权器未装配", nil
	}
	allowed, err = m.authorizer.Authorize(ctx, id.Subjects(), obj, act)
	if err != nil {
		return false, "RBAC 判定失败（fail-closed）", err
	}
	if !allowed {
		return false, "当前角色无权访问该资源", nil
	}
	return true, "", nil
}

// Wrap 是可复用的 HTTP 授权中间件；9108 由 OpsAuth.Wrap 在认证成功后调用 Decision，
// C 端若未来接入 RBAC 可直接复用本方法（保持 psSession 契约不变）。
func (m *Middleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := authn.FromContext(r.Context())
		allowed, reason, err := m.Decision(r.Context(), id, r.URL.Path, r.Method)
		if err != nil {
			m.log.ErrorContext(r.Context(), "rbac decision error", "audit", true,
				"subject", subjectOf(id), "obj", r.URL.Path, "act", r.Method, "err", err)
			writeAuthzError(w, http.StatusForbidden, errcode.Forbidden, reason)
			return
		}
		if !allowed {
			m.log.WarnContext(r.Context(), "rbac denied", "audit", true,
				"subject", subjectOf(id), "obj", r.URL.Path, "act", r.Method, "result", "denied", "reason", reason)
			if id == nil {
				w.Header().Set("WWW-Authenticate", "Bearer")
				writeAuthzError(w, http.StatusUnauthorized, errcode.Unauthorized, reason)
			} else {
				writeAuthzError(w, http.StatusForbidden, errcode.Forbidden, reason)
			}
			return
		}
		next.ServeHTTP(w, r)
	})
}

func subjectOf(id *authn.Identity) string {
	if id == nil {
		return "anonymous"
	}
	return id.Subject
}

func writeAuthzError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": code, "message": message})
}
