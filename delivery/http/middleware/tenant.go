package middleware

import (
	"net/http"

	kernelauth "github.com/agile-factory/kernel/kernel/auth"
	"github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/kernel/response"
)

// TenantContext returns a middleware that extracts the active tenant ID from the
// request and injects it into the context via kernelauth.WithTenantID.
//
// Resolution order (RFC-0003, section 4):
//  1. URL path param  : r.PathValue("tenantId")  — for routes like /api/v1/tenants/{tenantId}/...
//  2. X-Tenant-ID     : explicit header           — for programmatic clients
//  3. Query param     : ?tenantId=<uuid>          — fallback for simple integrations
//
// Custom domain / subdomain resolution (steps 1 and 2 of the full RFC-0003 flow)
// is deferred to v1.1 and will be implemented in a dedicated DNS-aware middleware.
//
// If no tenant ID can be resolved, the middleware responds with 400 Bad Request.
// Routes that do not require a tenant context (e.g. /health, /api/v1/auth/login)
// must be mounted BEFORE this middleware is applied, or use an explicit allow-list.
func TenantContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := resolveTenantID(r)
		if tenantID == "" {
			response.Error(w, errors.BadRequest("tenant context is required: provide tenantId in path, X-Tenant-ID header, or query param"))
			return
		}

		ctx := kernelauth.WithTenantID(r.Context(), tenantID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// resolveTenantID attempts to extract the tenant ID from the request using
// the resolution order defined in RFC-0003, section 4.
func resolveTenantID(r *http.Request) string {
	// 1. URL path parameter (highest priority — explicit and unambiguous)
	if id := r.PathValue("tenantId"); id != "" {
		return id
	}

	// 2. Explicit header (programmatic API clients)
	if id := r.Header.Get("X-Tenant-ID"); id != "" {
		return id
	}

	// 3. Query parameter (simple integrations, webhooks)
	if id := r.URL.Query().Get("tenantId"); id != "" {
		return id
	}

	return ""
}
