package http

import "net/http"

// RegisterRoutes registers all membership routes.
//
// RFC-0018 §6 — the POST /memberships/{id}/accept endpoint is removed; the
// invitee redeems an invite_token via POST /api/v1/auth/accept-invite which
// creates the user (if new) + ACTIVE membership atomically.
//
//   - POST /api/v1/memberships/invite           Invite (OWNER/ADMIN, by email)
//   - POST /api/v1/memberships/{id}/revoke      Revoke
//   - GET  /api/v1/memberships/tenant/{tenantId} List by tenant
//   - GET  /api/v1/memberships/user/{userId}     List by user
func RegisterRoutes(mux *http.ServeMux, h *MembershipHandler) {
	mux.HandleFunc("POST /api/v1/memberships/invite", h.Invite)
	mux.HandleFunc("POST /api/v1/memberships/{id}/revoke", h.Revoke)
	mux.HandleFunc("GET /api/v1/memberships/tenant/{tenantId}", h.ListByTenant)
	mux.HandleFunc("GET /api/v1/memberships/user/{userId}", h.ListByUser)
}
