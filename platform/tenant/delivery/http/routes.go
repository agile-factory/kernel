package http

import "net/http"

// RegisterRoutes mounts the tenant HTTP endpoints (RFC-0018 §3):
//
//   - GET    /api/v1/tenants/{id}                  membership required
//   - GET    /api/v1/tenants/slug/{slug}           membership required
//   - PATCH  /api/v1/tenants/{id}                  OWNER/ADMIN
//   - DELETE /api/v1/tenants/{id}                  OWNER (soft-delete)
//   - GET    /api/v1/tenants/{id}/settings         membership
//   - PATCH  /api/v1/tenants/{id}/settings         OWNER/ADMIN
//   - POST   /admin/v1/tenants                     platform super-admin
//
// The public POST /api/v1/tenants endpoint is intentionally absent — the
// only legitimate entry points for tenant creation are signup-owner (auth)
// and POST /admin/v1/tenants (super-admin sales-assisted onboarding).
func RegisterRoutes(mux *http.ServeMux, h *TenantHandler) {
	mux.HandleFunc("GET /api/v1/tenants/{id}", h.GetByID)
	mux.HandleFunc("GET /api/v1/tenants/slug/{slug}", h.GetBySlug)
	mux.HandleFunc("PATCH /api/v1/tenants/{id}", h.Update)
	mux.HandleFunc("DELETE /api/v1/tenants/{id}", h.Delete)

	mux.HandleFunc("GET /api/v1/tenants/{id}/settings", h.GetSettings)
	mux.HandleFunc("PATCH /api/v1/tenants/{id}/settings", h.UpdateSettings)

	mux.HandleFunc("POST /admin/v1/tenants", h.AdminCreate)
}
