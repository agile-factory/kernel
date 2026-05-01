package http

import "net/http"

// RegisterRoutes registers all tenant-related routes on the provided mux.
//
// Routes:
//
//	POST   /api/v1/tenants              → Create a new tenant
//	GET    /api/v1/tenants/{id}         → Get tenant by UUID
//	GET    /api/v1/tenants/slug/{slug}  → Get tenant by slug
func RegisterRoutes(mux *http.ServeMux, h *TenantHandler) {
	mux.HandleFunc("POST /api/v1/tenants", h.Create)
	mux.HandleFunc("GET /api/v1/tenants/{id}", h.GetByID)
	mux.HandleFunc("GET /api/v1/tenants/slug/{slug}", h.GetBySlug)
}
