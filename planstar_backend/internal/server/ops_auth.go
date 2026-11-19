package server

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"

	"planstar/internal/platform/authn"
	"planstar/internal/platform/conf"
	"planstar/internal/platform/observability"
	"planstar/internal/platform/pkg/errcode"
)

// anonymousActor 用于未通过鉴权或未配置令牌时的审计主体。
const anonymousActor = "anonymous"

// OpsAuth 保护运维端点：Bearer 令牌 + 来源 CIDR 白名单 + RBAC + 审计日志。
type OpsAuth struct {
	token   string
	actor   string
	allowed []netip.Prefix
	logger  *slog.Logger
	metrics *observability.Metrics

	// verifier / authorizer 由 ConfigureAuthn 注入（ADR-0007）。
	// 未注入时 Wrap 保持原有共享令牌行为，便于单测与灰度回滚。
	verifier   TokenVerifier
	authorizer Authorizer

	// warned 保证每个端点「鉴权关闭」的告警只打印一次。
	warned sync.Map
}

// NewOpsAuth 解析 CIDR 白名单并构造运维鉴权；CIDR 非法时启动即失败。
func NewOpsAuth(c conf.Ops, logger *slog.Logger, metrics *observability.Metrics) (*OpsAuth, error) {
	a := &OpsAuth{
		token:   strings.TrimSpace(c.Token),
		logger:  logger,
		metrics: metrics,
	}
	for _, raw := range c.AllowedCIDRs {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(raw)
		if err != nil {
			return nil, fmt.Errorf("ops.allowed_cidrs 不是合法的 CIDR: %q", raw)
		}
		a.allowed = append(a.allowed, prefix.Masked())
	}
	if a.token != "" {
		sum := sha256.Sum256([]byte(a.token))
		a.actor = "token:" + hex.EncodeToString(sum[:])[:8]
	} else {
		a.actor = anonymousActor
	}
	return a, nil
}

// Enabled 返回是否配置了运维令牌。
func (a *OpsAuth) Enabled() bool { return a.token != "" }

// AllowRemote 暴露 CIDR 判定给 /admin/auth/login 等不经 Wrap 的端点复用，
// 保证登录入口与运维端点受同一来源白名单约束。
func (a *OpsAuth) AllowRemote(remoteAddr string) bool { return a.remoteAllowed(remoteAddr) }

// Validate 校验环境相关约束：生产环境必须配置令牌，否则拒绝启动。
func (a *OpsAuth) Validate(env string) error {
	// 默认拒绝：除显式本地环境外，任何环境（prod/production/staging/docker…）
	// 未配置令牌都拒绝启动，避免环境命名差异绕过运维鉴权。
	if conf.IsProductionLikeEnv(env) && !a.Enabled() {
		return errors.New("非本地环境必须配置 PLANSTAR_OPS_TOKEN")
	}
	return nil
}

// TokenVerifier 校验个人账号 JWT（由 platform/authn.Issuer 实现）。
type TokenVerifier interface {
	VerifyAccess(ctx context.Context, raw string) (*authn.Identity, error)
}

// Authorizer 做 RBAC 判定（由 platform/authz.Enforcer 实现）。
type Authorizer interface {
	Authorize(ctx context.Context, subjects []string, obj, act string) (bool, error)
}

// ConfigureAuthn 装配运营域 JWT + RBAC（ADR-0007）。
// 未装配时 Wrap 保持原有「仅共享运维令牌」行为，便于单测与回滚；生产 wiring 必须装配。
func (a *OpsAuth) ConfigureAuthn(v TokenVerifier, az Authorizer) {
	a.verifier = v
	a.authorizer = az
}

// Wrap 返回带认证、RBAC 授权与审计的 handler；endpoint 用于指标与审计动作命名。
// 两个阶段严格分离：认证失败 401，越权 403；判定出错一律拒绝（fail-closed）。
func (a *OpsAuth) Wrap(endpoint string, next http.Handler) http.Handler {
	action := "ops." + endpoint + ".access"
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.Enabled() {
			// fail-closed：无令牌时直接拒绝，绝不「只打一条告警就放行」。
			a.warnDisabled(endpoint)
			a.incAuthFailed(endpoint)
			a.audit(r.Context(), anonymousActor, action, endpoint, "unauthorized", r.RemoteAddr, http.StatusUnauthorized)
			w.Header().Set("WWW-Authenticate", "Bearer")
			writeAdminJSON(w, http.StatusUnauthorized, map[string]string{
				"code":    errcode.Unauthorized,
				"message": "运维端点未配置凭据，已按最小权限原则关闭",
			})
			return
		}
		if !a.remoteAllowed(r.RemoteAddr) {
			a.incAuthFailed(endpoint)
			a.audit(r.Context(), anonymousActor, action, endpoint, "forbidden", r.RemoteAddr, http.StatusForbidden)
			writeAdminJSON(w, http.StatusForbidden, map[string]string{
				"code":    errcode.Forbidden,
				"message": "来源地址不允许",
			})
			return
		}
		token, ok := bearerToken(r)
		if !ok {
			a.unauthorized(r, w, endpoint, action, "缺少运维凭据")
			return
		}

		// 阶段一：共享运维令牌（显式 superadmin 策略）或个人账号 JWT。
		var identity *authn.Identity
		switch {
		case subtle.ConstantTimeCompare([]byte(token), []byte(a.token)) == 1:
			identity = &authn.Identity{
				Subject: "ops:break-glass", Roles: []string{authn.RoleSuperAdmin},
				Type: "ops", ViaBreakGlass: true,
			}
		case a.verifier != nil:
			id, err := a.verifier.VerifyAccess(r.Context(), token)
			if err != nil {
				a.unauthorized(r, w, endpoint, action, jwtFailureMessage(err))
				return
			}
			identity = id
		default:
			a.unauthorized(r, w, endpoint, action, "缺少或无效的运维凭据")
			return
		}

		// 阶段二：Casbin RBAC。策略里显式声明 superadmin，代码里没有绕过分支。
		if a.authorizer != nil {
			allowed, err := a.authorizer.Authorize(r.Context(), identity.Subjects(), r.URL.Path, r.Method)
			if err != nil || !allowed {
				a.incAuthFailed(endpoint)
				a.audit(r.Context(), actorOf(a, identity), action, endpoint, "forbidden", r.RemoteAddr, http.StatusForbidden)
				a.logger.WarnContext(r.Context(), "rbac denied", "audit", true,
					"subject", identity.Subject, "obj", r.URL.Path, "act", r.Method,
					"break_glass", identity.ViaBreakGlass)
				writeAdminJSON(w, http.StatusForbidden, map[string]string{
					"code":    errcode.Forbidden,
					"message": "当前角色无权访问该资源",
				})
				return
			}
			// ADR-0010 P0①：鉴权通过同样产出审计记录，与拒绝侧对称（补 PolicyEvaluationLog 闭环）。
			a.logger.InfoContext(r.Context(), "rbac allowed", "audit", true,
				"subject", identity.Subject, "obj", r.URL.Path, "act", r.Method,
				"break_glass", identity.ViaBreakGlass)
		}

		r = r.WithContext(authn.WithIdentity(r.Context(), identity))
		sw := &auditStatusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		a.audit(r.Context(), actorOf(a, identity), action, endpoint, statusResult(sw.status), r.RemoteAddr, sw.status)
	})
}

func actorOf(a *OpsAuth, id *authn.Identity) string {
	if id == nil || id.ViaBreakGlass {
		return a.actor
	}
	return id.Subject
}

// jwtFailureMessage 只返回可安全外显的分类信息，不泄漏密钥、Redis 地址等内部细节。
func jwtFailureMessage(err error) string {
	switch {
	case errors.Is(err, authn.ErrTokenExpired):
		return "访问令牌已过期"
	case errors.Is(err, authn.ErrTokenRevoked):
		return "访问令牌已撤销"
	case errors.Is(err, authn.ErrTokenInvalid), errors.Is(err, authn.ErrWrongTokenType):
		return "访问令牌无效"
	default:
		return "访问令牌校验失败（fail-closed）"
	}
}

func (a *OpsAuth) unauthorized(r *http.Request, w http.ResponseWriter, endpoint, action, message string) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	a.incAuthFailed(endpoint)
	a.audit(r.Context(), anonymousActor, action, endpoint, "unauthorized", r.RemoteAddr, http.StatusUnauthorized)
	writeAdminJSON(w, http.StatusUnauthorized, map[string]string{
		"code": errcode.Unauthorized, "message": message,
	})
}

// auditWrite 记录一次运营写操作（举报处置、企划审核）；调用方负责给出业务结果。
func (a *OpsAuth) auditWrite(ctx context.Context, action, target, result string) {
	a.audit(ctx, a.actor, action, target, result, "", 0)
}

// audit 输出结构化审计日志并累加审计指标；任何分支都不会记录令牌明文。
func (a *OpsAuth) audit(ctx context.Context, actor, action, target, result, remoteAddr string, status int) {
	if a.metrics != nil {
		a.metrics.IncOpsAudit(action, result)
	}
	if a.logger == nil {
		return
	}
	a.logger.InfoContext(ctx, "ops audit",
		"audit", true,
		"actor", actor,
		"action", action,
		"target", target,
		"result", result,
		"remote_addr", remoteAddr,
		"status", status,
	)
}

func (a *OpsAuth) incAuthFailed(endpoint string) {
	if a.metrics != nil {
		a.metrics.IncOpsAuthFailed(endpoint)
	}
}

// warnDisabled 在未配置令牌时按端点告警一次，避免刷屏。
func (a *OpsAuth) warnDisabled(endpoint string) {
	if _, loaded := a.warned.LoadOrStore(endpoint, struct{}{}); loaded {
		return
	}
	if a.logger != nil {
		a.logger.Warn("ops auth disabled", "endpoint", endpoint)
	}
}

// remoteAllowed 判断来源地址是否命中白名单；白名单为空时放行。
func (a *OpsAuth) remoteAllowed(remoteAddr string) bool {
	if len(a.allowed) == 0 {
		return true
	}
	addr, ok := remoteIP(remoteAddr)
	if !ok {
		return false
	}
	for _, prefix := range a.allowed {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

// remoteIP 从 RemoteAddr 提取 IP（忽略 X-Forwarded-For，避免被伪造头绕过）。
func remoteIP(remoteAddr string) (netip.Addr, bool) {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	addr, err := netip.ParseAddr(strings.TrimSpace(host))
	if err != nil {
		return netip.Addr{}, false
	}
	return addr.Unmap(), true
}

// bearerToken 解析 `Authorization: Bearer <token>`。
func bearerToken(r *http.Request) (string, bool) {
	parts := strings.Fields(r.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	return parts[1], true
}

// statusResult 把 HTTP 状态码归并为审计结果类别。
func statusResult(status int) string {
	switch {
	case status >= 500:
		return "error"
	case status >= 400:
		return "client_error"
	default:
		return "ok"
	}
}

// auditStatusWriter 仅捕获状态码，供运维写接口审计使用。
type auditStatusWriter struct {
	http.ResponseWriter
	status int
}

func (w *auditStatusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
