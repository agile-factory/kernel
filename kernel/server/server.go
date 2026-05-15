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
	"net/http"

	"github.com/AgileFactory/kernel/kernel/config"
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
		},
	}
}

// Start runs the HTTP server.
func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
