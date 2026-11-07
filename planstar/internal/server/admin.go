package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	khttp "github.com/go-kratos/kratos/v3/transport/http"

	"planstar/internal/biz"
	"planstar/internal/conf"
	"planstar/internal/observability"
	"planstar/internal/ops"
)

// NewProbeHandlers 返回 /healthz 与 /readyz 的 handler（admin 与测试复用同一套语义）。
// healthz 只表示进程存活，不检查依赖；readyz 由 ops.Readiness 决定 200/503。
func NewProbeHandlers(ready *ops.Readiness, version string, metrics *observability.Metrics) (http.Handler, http.Handler) {
	live := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ready.Live()
		if version != "" {
			resp.Version = version
		}
		writeProbeJSON(w, http.StatusOK, resp)
	})
	readyz := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ready.Check(r.Context())
		if metrics != nil {
			metrics.ObserveProbe("readyz", resp.Checks, resp.Cached)
		}
		status := http.StatusOK
		if !resp.Ready {
			status = http.StatusServiceUnavailable
		}
		writeProbeJSON(w, status, resp)
	})
	return live, readyz
}

func writeProbeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// NewAdminServer 提供运维端点：/metrics、/healthz、/readyz，以及运营后台的举报审核与企划审核端点。
// 与业务端口分离，避免指标与审核能力暴露在公网 API 上。
func NewAdminServer(c conf.Server, metrics *observability.Metrics, ready *ops.Readiness, version string, auth *OpsAuth, reports *biz.ReportUsecase, projects *biz.ProjectUsecase, logger *slog.Logger) *khttp.Server {
	mux := http.NewServeMux()
	// /metrics 与运营后台需要 Bearer 凭据；探针保持匿名，供容器编排健康检查使用。
	mux.Handle("/metrics", auth.Wrap("metrics", metrics.Handler()))
	live, readyz := NewProbeHandlers(ready, version, metrics)
	mux.Handle("/healthz", live)
	mux.Handle("/readyz", readyz)

	// 运营后台单页：举报处理台 + 企划审核。
	mux.Handle("GET /admin/console", auth.Wrap("admin", http.HandlerFunc(adminConsole)))
	mux.Handle("GET /admin", auth.Wrap("admin", http.HandlerFunc(adminConsole)))
	mux.Handle("GET /admin/", auth.Wrap("admin", http.HandlerFunc(adminConsole)))

	// 举报处理台：列表 + 处置（仅运营内网端口可达）。
	mux.Handle("GET /admin/reports", auth.Wrap("admin", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		items, err := reports.List(r.Context(), r.URL.Query().Get("status"), limit)
		if err != nil {
			writeAdminError(w, err)
			return
		}
		writeAdminJSON(w, http.StatusOK, map[string]any{
			"items": items,
			"total": len(items),
		})
	})))
	mux.Handle("POST /admin/reports/{id}/handle", auth.Wrap("admin", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reportID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			auth.auditWrite(r.Context(), "report.handle", r.PathValue("id"), "bad_request")
			writeAdminError(w, biz.ErrBadRequest)
			return
		}
		target := strconv.FormatInt(reportID, 10)
		var body struct {
			Status string `json:"status"`
			Note   string `json:"note"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			auth.auditWrite(r.Context(), "report.handle", target, "bad_request")
			writeAdminError(w, biz.ErrBadRequest)
			return
		}
		item, err := reports.Handle(r.Context(), reportID, body.Status, body.Note)
		if err != nil {
			switch {
			case errors.Is(err, biz.ErrBadRequest):
				auth.auditWrite(r.Context(), "report.handle", target, "bad_request")
			case errors.Is(err, biz.ErrNotFound):
				auth.auditWrite(r.Context(), "report.handle", target, "not_found")
			default:
				auth.auditWrite(r.Context(), "report.handle", target, "error")
			}
			writeAdminError(w, err)
			return
		}
		auth.auditWrite(r.Context(), "report.handle", target, "success")
		writeAdminJSON(w, http.StatusOK, item)
	})))

	// 企划审核：查看黑名单判定结果 + 通过/驳回。发布动作只在此处发生，
	// approve 前强制通过品类黑名单校验（命中即 400，并返回 reasons）。
	mux.Handle("GET /admin/projects/{id}", auth.Wrap("admin", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		projectID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || projectID <= 0 {
			writeAdminError(w, biz.ErrBadRequest)
			return
		}
		mod, err := projects.Moderation(r.Context(), projectID)
		if err != nil {
			writeAdminError(w, err)
			return
		}
		writeAdminJSON(w, http.StatusOK, mod)
	})))
	mux.Handle("POST /admin/projects/{id}/review", auth.Wrap("admin", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		projectID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || projectID <= 0 {
			auth.auditWrite(r.Context(), "project.review", r.PathValue("id"), "error")
			writeAdminError(w, biz.ErrBadRequest)
			return
		}
		target := strconv.FormatInt(projectID, 10)
		var body struct {
			Decision string `json:"decision"`
		}
		r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			auth.auditWrite(r.Context(), "project.review", target, "error")
			writeAdminError(w, biz.ErrBadRequest)
			return
		}
		mod, err := projects.ReviewProject(r.Context(), projectID, body.Decision)
		if err != nil {
			var blocked *biz.BlacklistBlockedError
			if errors.As(err, &blocked) {
				auth.auditWrite(r.Context(), "project.review", target, "blocked")
				writeAdminJSON(w, http.StatusBadRequest, map[string]any{
					"code":    "BAD_REQUEST",
					"message": blocked.Error(),
					"reasons": blocked.Hits,
				})
				return
			}
			auth.auditWrite(r.Context(), "project.review", target, "error")
			writeAdminError(w, err)
			return
		}
		auth.auditWrite(r.Context(), "project.review", target, body.Decision)
		writeAdminJSON(w, http.StatusOK, mod)
	})))

	srv := khttp.NewServer(
		khttp.Address(c.Admin.Addr),
		khttp.Timeout(10*time.Second),
	)
	// 运维端口只服务轻量探针与审核接口，收紧读写与空闲超时，避免慢连接占用。
	srv.Server.ReadHeaderTimeout = 5 * time.Second
	srv.Server.ReadTimeout = 10 * time.Second
	srv.Server.WriteTimeout = 10 * time.Second
	srv.Server.IdleTimeout = 60 * time.Second
	// 整站套指标/追踪中间件，让审计日志也能带上 trace_id；探针同样匿名可达。
	srv.HandlePrefix("/", metrics.Middleware(recoverer(logger, mux), logger))
	return srv
}

func writeAdminJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeAdminError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, biz.ErrBadRequest):
		writeAdminJSON(w, http.StatusBadRequest, map[string]string{"code": "BAD_REQUEST", "message": "参数错误"})
	case errors.Is(err, biz.ErrNotFound):
		writeAdminJSON(w, http.StatusNotFound, map[string]string{"code": "NOT_FOUND", "message": "举报单不存在"})
	default:
		writeAdminJSON(w, http.StatusInternalServerError, map[string]string{"code": "INTERNAL", "message": "服务异常"})
	}
}
