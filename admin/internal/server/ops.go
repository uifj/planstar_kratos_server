package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"planstar/admin/internal/conf"
)

const defaultOpsAddr = "127.0.0.1:9110"

// ReadinessChecker is the storage-level readiness probe. *data.Data satisfies
// it; the interface lives here so the server package does not import data.
type ReadinessChecker interface {
	Ready(context.Context) error
}

// OpsServer is a standalone net/http server for /metrics, /healthz and
// /readyz. It is exposed as a Kratos transport.Server so the app lifecycle
// starts and gracefully stops it alongside HTTP and gRPC.
type OpsServer struct {
	srv     *http.Server
	ready   ReadinessChecker
	metrics *Metrics
}

// NewOpsServer builds the ops listener. It binds to server.admin.addr and
// defaults to loopback so probes and metrics never reach the public port.
func NewOpsServer(c *conf.Server, ready ReadinessChecker, metrics *Metrics) *OpsServer {
	addr := c.GetAdmin().GetAddr()
	if addr == "" {
		addr = defaultOpsAddr
	}

	mux := http.NewServeMux()
	if metrics != nil {
		mux.Handle("/metrics", metrics.Handler())
	}
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeProbe(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if ready == nil {
			if metrics != nil {
				metrics.SetReady(true)
			}
			writeProbe(w, http.StatusOK, map[string]string{"status": "ok"})
			return
		}
		if err := ready.Ready(r.Context()); err != nil {
			if metrics != nil {
				metrics.SetReady(false)
			}
			writeProbe(w, http.StatusServiceUnavailable, map[string]string{
				"status": "unavailable",
				"error":  err.Error(),
			})
			return
		}
		if metrics != nil {
			metrics.SetReady(true)
		}
		writeProbe(w, http.StatusOK, map[string]string{"status": "ready"})
	})

	return &OpsServer{
		srv: &http.Server{
			Addr:              addr,
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       10 * time.Second,
			WriteTimeout:      10 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
		ready:   ready,
		metrics: metrics,
	}
}

// Handler exposes the mux for tests and embedding.
func (s *OpsServer) Handler() http.Handler {
	return s.srv.Handler
}

// Start blocks until the listener stops, matching the Kratos server contract.
func (s *OpsServer) Start(context.Context) error {
	err := s.srv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Stop gracefully drains in-flight probe requests.
func (s *OpsServer) Stop(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

func writeProbe(w http.ResponseWriter, status int, body map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
