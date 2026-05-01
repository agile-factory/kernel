package http

import "net/http"

// RegisterRoutes registers all audit-related routes on the provided mux.
//
// Routes:
//
//	GET /api/v1/tenants/{tenantId}/audit-logs  → List audit logs for a tenant (ROLE_ADMIN only)
func RegisterRoutes(mux *http.ServeMux, h *AuditHandler) {
	mux.HandleFunc("GET /api/v1/tenants/{tenantId}/audit-logs", h.ListByTenant)
}
