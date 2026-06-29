// Package health provides liveness and readiness HTTP handlers.
//
// Usage:
//
//	checker := health.New(health.Config{
//	    Version:  "1.0.0",
//	    DBPinger: pool,
//	})
//	mux.HandleFunc("GET /health", checker.LivenessHandler)
//	mux.HandleFunc("GET /ready", checker.ReadinessHandler)
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"
)

// Pinger is satisfied by *pgxpool.Pool (Ping method).
type Pinger interface {
	Ping(ctx context.Context) error
}

// Config captures health check dependencies.
type Config struct {
	// Version is the application version string (injected at build time).
	Version string
	// DBPinger is used by the readiness check. May be nil to skip DB check.
	DBPinger Pinger
}

// Checker holds the handlers.
type Checker struct {
	cfg   Config
	start time.Time
	ready atomic.Bool
}

// New creates a Checker. Call MarkReady() once the application has finished
// startup (migrations applied, external dependencies reachable).
func New(cfg Config) *Checker {
	return &Checker{cfg: cfg, start: time.Now()}
}

// MarkReady signals that the application is ready to serve traffic.
func (c *Checker) MarkReady() { c.ready.Store(true) }

// livenessResponse is the JSON body for GET /health.
type livenessResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Uptime  string `json:"uptime"`
}

// LivenessHandler handles GET /health.
// Always returns 200 if the process is running.
func (c *Checker) LivenessHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(livenessResponse{
		Status:  "ok",
		Version: c.cfg.Version,
		Uptime:  time.Since(c.start).Round(time.Second).String(),
	})
}

// readinessResponse is the JSON body for GET /ready.
type readinessResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

// ReadinessHandler handles GET /ready.
// Returns 200 when ready; 503 otherwise.
func (c *Checker) ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	checks := map[string]string{}
	allOK := true

	if c.cfg.DBPinger != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := c.cfg.DBPinger.Ping(ctx); err != nil {
			checks["database"] = "error: " + err.Error()
			allOK = false
		} else {
			checks["database"] = "ok"
		}
	}

	status := "ok"
	code := http.StatusOK
	if !allOK || !c.ready.Load() {
		status = "unavailable"
		code = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(readinessResponse{
		Status: status,
		Checks: checks,
	})
}
