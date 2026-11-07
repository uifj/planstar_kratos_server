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

	"planstar/internal/conf"
	"planstar/internal/observability"
)

// anonymousActor 用于未通过鉴权或未配置令牌时的审计主体。
const anonymousActor = "anonymous"

// OpsAuth 保护运维端点：Bearer 令牌 + 来源 CIDR 白名单 + 审计日志。
type OpsAuth struct {
	token   string
	actor   string
	allowed []netip.Prefix
	logger  *slog.Logger
	metrics *observability.Metrics

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

// Validate 校验环境相关约束：生产环境必须配置令牌，否则拒绝启动。
func (a *OpsAuth) Validate(env string) error {
	if strings.EqualFold(strings.TrimSpace(env), "prod") && !a.Enabled() {
		return errors.New("生产环境必须配置 PLANSTAR_OPS_TOKEN")
	}
	return nil
}

// Wrap 返回带鉴权与审计的 handler；endpoint 用于指标与审计动作命名。
func (a *OpsAuth) Wrap(endpoint string, next http.Handler) http.Handler {
	action := "ops." + endpoint + ".access"
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.Enabled() {
			a.warnDisabled(endpoint)
		} else {
			if !a.remoteAllowed(r.RemoteAddr) {
				a.incAuthFailed(endpoint)
				a.audit(r.Context(), anonymousActor, action, endpoint, "forbidden", r.RemoteAddr, http.StatusForbidden)
				writeAdminJSON(w, http.StatusForbidden, map[string]string{
					"code":    "FORBIDDEN",
					"message": "来源地址不允许",
				})
				return
			}
			token, ok := bearerToken(r)
			if !ok || subtle.ConstantTimeCompare([]byte(token), []byte(a.token)) != 1 {
				w.Header().Set("WWW-Authenticate", "Bearer")
				a.incAuthFailed(endpoint)
				a.audit(r.Context(), anonymousActor, action, endpoint, "unauthorized", r.RemoteAddr, http.StatusUnauthorized)
				writeAdminJSON(w, http.StatusUnauthorized, map[string]string{
					"code":    "UNAUTHORIZED",
					"message": "缺少或无效的运维凭据",
				})
				return
			}
		}

		sw := &auditStatusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		a.audit(r.Context(), a.actor, action, endpoint, statusResult(sw.status), r.RemoteAddr, sw.status)
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
