package http

import (
	"log/slog"
	"net/http"

	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/delivery/http/middleware"
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
//
// Route-level middlewares (Auth, TenantContext) are applied selectively
// via ApplyAuthMiddleware and ApplyTenantMiddleware helpers below.
func ApplyGlobalMiddlewares(handler http.Handler, log *slog.Logger) http.Handler {
	h := handler
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

