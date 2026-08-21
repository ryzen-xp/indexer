// Package httpserver provides an opt-in HTTP server exposing Prometheus
// metrics and a liveness/readiness health check for the running indexer
// process, so it can be monitored and probed by Docker/k8s.
package httpserver

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/miguelnietoa/stellar-explorer/indexer/internal/health"
	"github.com/miguelnietoa/stellar-explorer/indexer/internal/metrics"
)

// pingTimeout bounds how long /healthz waits on the database before
// reporting unhealthy, so a stuck connection can't hang the probe.
const pingTimeout = 2 * time.Second

// pipelineStaleAfter is how long the ingestion loop can go without
// completing a poll cycle before /healthz reports it as not advancing.
// Ledgers close every ~5s, so this comfortably tolerates transient RPC
// slowness without masking a genuinely stuck pipeline.
const pipelineStaleAfter = 2 * time.Minute

// dbPinger is the subset of *sql.DB that /healthz needs, kept as an
// interface so the handler can be exercised with a fake in tests.
type dbPinger interface {
	PingContext(ctx context.Context) error
}

// staleChecker reports whether the live pipeline has stopped advancing.
// Kept as a function value (defaulting to health.Stale) so tests can
// simulate a stuck pipeline without depending on real elapsed time.
type staleChecker func(maxAge time.Duration) bool

// Server serves /metrics, /healthz, and the domains read API.
type Server struct {
	srv     *http.Server
	domains DomainReader
}

// New builds a Server listening on addr. db is used by /healthz to verify
// the database is reachable and confirm the live pipeline is still
// advancing.
func New(addr string, db dbPinger) *Server {
	s := &Server{}
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(metrics.Registry, promhttp.HandlerOpts{}))
	mux.HandleFunc("/healthz", healthzHandler(db, health.Stale))
	mux.HandleFunc("GET /v1/domains/{name}/events", s.handleDomainEvents)
	mux.HandleFunc("GET /v1/domains/{name}", s.handleDomainByName)
	mux.HandleFunc("GET /v1/domains", s.handleDomains)

	s.srv = &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	return s
}

// SetDomainReader attaches the domains store used by the read API. When unset,
// domain endpoints return HTTP 200 with indexed=false.
func (s *Server) SetDomainReader(r DomainReader) {
	s.domains = r
}

// Start blocks serving requests until the server is shut down. It returns
// nil on a clean shutdown (http.ErrServerClosed).
func (s *Server) Start() error {
	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

type healthStatus struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

// healthzHandler reports readiness based on database connectivity and
// pipeline liveness: 200 when the database responds and the ingestion loop
// has ticked within pipelineStaleAfter, 503 otherwise. The underlying DB
// error is logged server-side but never returned in the response body, since
// /healthz may be reachable to unauthenticated callers (probes, load
// balancers) and the error can carry internal infrastructure details.
func healthzHandler(db dbPinger, isStale staleChecker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), pingTimeout)
		defer cancel()

		w.Header().Set("Content-Type", "application/json")

		if err := db.PingContext(ctx); err != nil {
			log.Printf("healthz: database ping failed: %v", err)
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(healthStatus{Status: "unhealthy", Reason: "database unreachable"})
			return
		}

		if isStale(pipelineStaleAfter) {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(healthStatus{Status: "unhealthy", Reason: "pipeline not advancing"})
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(healthStatus{Status: "ok"})
	}
}
