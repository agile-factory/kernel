// Package server is a thin wrapper around *http.Server. It exists so the
// example main.go (and consumer mains) stay short — Bind address comes from
// config.Config, and Start / Shutdown match the lifecycle the rest of the
// kernel expects.
//
// Graceful shutdown is the caller's responsibility — wire a signal
// listener and call Shutdown(ctx) with a deadline; the example in
// examples/saas-b2b/main.go shows the canonical pattern.
package server

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/agile-factory/kernel/kernel/config"
)

// Server is a wrapper around http.Server
type Server struct {
	httpServer *http.Server
}

// New initializes a new Server instance.
func New(cfg *config.Config, handler http.Handler) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:    ":" + cfg.Port,
			Handler: handler,
			// Without these a client can hold a connection open forever
			// (Slowloris). WriteTimeout sits above DefaultRequestTimeout so
			// TimeoutMiddleware answers first.
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      DefaultRequestTimeout + 30*time.Second,
			IdleTimeout:       120 * time.Second,
		},
	}
}

// Start runs the HTTP server. It returns nil after a graceful Shutdown and
// an error only when the server could not start or stopped abnormally.
func (s *Server) Start() error {
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown gracefully shuts down the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
