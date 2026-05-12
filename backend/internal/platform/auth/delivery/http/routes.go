package http

import (
	"net/http"
)

// RegisterRoutes registers the auth HTTP endpoints to the provided multiplexer.
func RegisterRoutes(mux *http.ServeMux, handler *AuthHandler) {
	mux.HandleFunc("POST /api/v1/auth/register", handler.Register)
	mux.HandleFunc("POST /api/v1/auth/login", handler.Login)
}
