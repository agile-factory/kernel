package http

import "net/http"

// RegisterRoutes registers all audit-related routes on the provided mux.
//
// v1 routes (offset-based — stable, unchanged):
//
//	GET /api/v1/tenants/{tenantId}/audit-logs  → ListByTenant
//	GET /api/v1/audit-logs                     → ListAll
//
// v2 routes (cursor-based — added in v2):
//
//	GET /api/v2/tenants/{tenantId}/audit-logs  → ListByTenantV2
//	GET /api/v2/audit-logs                     → ListAllV2
func RegisterRoutes(mux *http.ServeMux, h *AuditHandler, hv2 *AuditHandlerV2) {
	// ── v1 — offset-based (kept for backward compatibility) ──────────────────
	mux.HandleFunc("GET /api/v1/tenants/{tenantId}/audit-logs", h.ListByTenant)
	mux.HandleFunc("GET /api/v1/audit-logs", h.ListAll)

	// ── v2 — cursor-based ────────────────────────────────────────────────────
	mux.HandleFunc("GET /api/v2/tenants/{tenantId}/audit-logs", hv2.ListByTenantV2)
	mux.HandleFunc("GET /api/v2/audit-logs", hv2.ListAllV2)
}
