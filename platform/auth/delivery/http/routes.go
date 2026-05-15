package http

import "net/http"

// RegisterRoutes registers the auth HTTP endpoints to the provided mux.
//
// RFC-0018 — /api/v1/auth/register is intentionally absent (B2B onboarding
// is invite-only; the first user of a workspace lands via /signup-owner).
//
//	POST /api/v1/auth/signup-owner   → SignupOwner
//	POST /api/v1/auth/accept-invite  → AcceptInvite
//	POST /api/v1/auth/verify-email   → VerifyEmail
//	POST /api/v1/auth/login          → Login
//	POST /api/v1/auth/refresh        → Refresh (rolling rotation)
//	POST /api/v1/auth/logout         → Logout
func RegisterRoutes(mux *http.ServeMux, handler *AuthHandler) {
	mux.HandleFunc("POST /api/v1/auth/signup-owner", handler.SignupOwner)
	mux.HandleFunc("POST /api/v1/auth/accept-invite", handler.AcceptInvite)
	mux.HandleFunc("POST /api/v1/auth/verify-email", handler.VerifyEmail)
	mux.HandleFunc("POST /api/v1/auth/login", handler.Login)
	mux.HandleFunc("POST /api/v1/auth/refresh", handler.Refresh)
	mux.HandleFunc("POST /api/v1/auth/logout", handler.Logout)
}
