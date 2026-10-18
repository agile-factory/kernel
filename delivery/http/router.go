// Package http hosts the HTTP middlewares shared by every platform module
// (Recovery, Logger, Auth, TenantContext) plus thin helpers to compose them.
//
// Routes themselves are NOT registered here. Each platform module exposes
// its own RegisterRoutes(mux) and the consumer main composes them onto a
// single mux. The helpers in this package layer cross-cutting concerns
// (panic recovery, structured access logs) on top of that mux.
//
// Middleware ordering matters. ApplyGlobalMiddlewares applies, from outermost
// to innermost: Recovery → Logger → handler. Route-level middlewares
// (WithAuth, WithTenant) are added selectively by individual handlers.
// Request bodies are capped at 1 MiB (middleware.MaxBytes).
package http

import (
	"log/slog"
	"net/http"

	"github.com/agile-factory/kernel/delivery/http/middleware"
)

// NewRouter creates and configures the main HTTP router with all global middlewares.
func NewRouter(log *slog.Logger) *http.ServeMux {
	mux := http.NewServeMux()

	return mux
}

// ApplyGlobalMiddlewares wraps the main router with global middlewares.
// The order of middlewares is important. They are executed from outside in:
//
//  1. Recovery    — outermost, catches panics from all layers below
//  2. Logger      — logs every request with status, method, path, duration
//  3. MaxBytes    — caps request bodies at DefaultMaxBodyBytes (1 MiB)
//
// Route-level middlewares (Auth, TenantContext) are applied selectively
// via ApplyAuthMiddleware and ApplyTenantMiddleware helpers below.
func ApplyGlobalMiddlewares(handler http.Handler, log *slog.Logger) http.Handler {
	h := handler
	h = middleware.MaxBytes(middleware.DefaultMaxBodyBytes)(h)
	h = middleware.Logger(log)(h)
	h = middleware.Recovery(log)(h)
	return h
}

// WithAuth wraps a handler with JWT authentication.
// The authenticated user ID is injected into the request context.
// Use this for routes that require an authenticated user.
func WithAuth(handler http.Handler, jwtSecret string) http.Handler {
	return middleware.Auth(jwtSecret)(handler)
}

// WithTenant wraps a handler with tenant context resolution.
// The active tenant ID is injected into the request context.
// Must be applied after WithAuth when both are needed.
func WithTenant(handler http.Handler) http.Handler {
	return middleware.TenantContext(handler)
}

