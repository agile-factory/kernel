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
// The order of middlewares is important. They are executed from outside in.
func ApplyGlobalMiddlewares(handler http.Handler, log *slog.Logger) http.Handler {
	// Start with the base handler
	h := handler

	// Apply Logger middleware
	h = middleware.Logger(log)(h)

	// Apply Recovery middleware (outermost, so it catches panics in Logger or downstream)
	h = middleware.Recovery(log)(h)

	return h
}
