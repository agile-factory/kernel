package http

import "net/http"

// RegisterRoutes registers all membership-related routes on the provided mux.
//
// Routes:
//
//	POST /api/v1/memberships/invite           → Invite a user to a tenant (admin only)
//	POST /api/v1/memberships/{id}/accept      → Accept an invitation
//	POST /api/v1/memberships/{id}/revoke      → Revoke a membership (admin only)
//	GET  /api/v1/memberships/tenant/{tenantId} → List all members of a tenant
//	GET  /api/v1/users/{userId}/memberships   → List all tenants a user belongs to
func RegisterRoutes(mux *http.ServeMux, h *MembershipHandler) {
	mux.HandleFunc("POST /api/v1/memberships/invite", h.Invite)
	mux.HandleFunc("POST /api/v1/memberships/{id}/accept", h.Accept)
	mux.HandleFunc("POST /api/v1/memberships/{id}/revoke", h.Revoke)
	mux.HandleFunc("GET /api/v1/memberships/tenant/{tenantId}", h.ListByTenant)
	mux.HandleFunc("GET /api/v1/memberships/user/{userId}", h.ListByUser)
}

