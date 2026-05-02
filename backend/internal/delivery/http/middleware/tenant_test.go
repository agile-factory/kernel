package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	kernelauth "github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/kernel/auth"
	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/delivery/http/middleware"
)

// ─── tenantCapture records the tenant ID injected into the context ────────────

type tenantCapture struct {
	called   bool
	tenantID string
}

func (h *tenantCapture) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.called = true
	h.tenantID = kernelauth.TenantIDFromContext(r.Context())
	w.WriteHeader(http.StatusOK)
}

// ─── TenantContext — path parameter (highest priority) ────────────────────────

func TestTenantContext_PathParam_InjectsTenantID(t *testing.T) {
	next := &tenantCapture{}
	handler := middleware.TenantContext(next)

	// Simulate a ServeMux path-value by using httptest and a mux that sets PathValue.
	// Since net/http.Request.PathValue is populated by the stdlib mux routing,
	// we simulate it via a wrapping handler that calls r.SetPathValue before passing
	// the request to the middleware.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/tenant-path-001/audit-logs", nil)
	setPathValue(req, "tenantId", "tenant-path-001")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusOK)
	}
	if !next.called {
		t.Error("next handler should have been called")
	}
	if next.tenantID != "tenant-path-001" {
		t.Errorf("tenantID from context: got %q, want %q", next.tenantID, "tenant-path-001")
	}
}

// ─── TenantContext — X-Tenant-ID header ──────────────────────────────────────

func TestTenantContext_Header_InjectsTenantID(t *testing.T) {
	next := &tenantCapture{}
	handler := middleware.TenantContext(next)

	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	req.Header.Set("X-Tenant-ID", "tenant-header-002")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusOK)
	}
	if next.tenantID != "tenant-header-002" {
		t.Errorf("tenantID from context: got %q, want %q", next.tenantID, "tenant-header-002")
	}
}

// ─── TenantContext — query parameter ─────────────────────────────────────────

func TestTenantContext_QueryParam_InjectsTenantID(t *testing.T) {
	next := &tenantCapture{}
	handler := middleware.TenantContext(next)

	req := httptest.NewRequest(http.MethodGet, "/resource?tenantId=tenant-query-003", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusOK)
	}
	if next.tenantID != "tenant-query-003" {
		t.Errorf("tenantID from context: got %q, want %q", next.tenantID, "tenant-query-003")
	}
}

// ─── TenantContext — missing tenant — 400 ────────────────────────────────────

func TestTenantContext_NoTenantID_Returns400(t *testing.T) {
	next := &tenantCapture{}
	handler := middleware.TenantContext(next)

	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if next.called {
		t.Error("next handler must not be called when tenant ID is missing")
	}
}

func TestTenantContext_NoTenantID_ReturnsJSONError(t *testing.T) {
	handler := middleware.TenantContext(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type: got %q, want %q", ct, "application/json")
	}
}

// ─── TenantContext — resolution priority ─────────────────────────────────────

// When path param and header are both present, path param takes precedence.
func TestTenantContext_PathParamTakesPrecedenceOverHeader(t *testing.T) {
	next := &tenantCapture{}
	handler := middleware.TenantContext(next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/path-tenant/resource", nil)
	setPathValue(req, "tenantId", "path-tenant")
	req.Header.Set("X-Tenant-ID", "header-tenant")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if next.tenantID != "path-tenant" {
		t.Errorf("expected path param to win over header: got %q", next.tenantID)
	}
}

// When path param and query param are both present, path param takes precedence.
func TestTenantContext_PathParamTakesPrecedenceOverQuery(t *testing.T) {
	next := &tenantCapture{}
	handler := middleware.TenantContext(next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/path-tenant/resource?tenantId=query-tenant", nil)
	setPathValue(req, "tenantId", "path-tenant")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if next.tenantID != "path-tenant" {
		t.Errorf("expected path param to win over query: got %q", next.tenantID)
	}
}

// When header and query param are both present, header takes precedence.
func TestTenantContext_HeaderTakesPrecedenceOverQuery(t *testing.T) {
	next := &tenantCapture{}
	handler := middleware.TenantContext(next)

	req := httptest.NewRequest(http.MethodGet, "/resource?tenantId=query-tenant", nil)
	req.Header.Set("X-Tenant-ID", "header-tenant")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if next.tenantID != "header-tenant" {
		t.Errorf("expected header to win over query: got %q", next.tenantID)
	}
}

// ─── TenantContext — context propagation ─────────────────────────────────────

func TestTenantContext_TenantIDStoredInContext(t *testing.T) {
	var captured string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = kernelauth.TenantIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.TenantContext(next)
	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	req.Header.Set("X-Tenant-ID", "tenant-ctx-check")

	handler.ServeHTTP(httptest.NewRecorder(), req)

	if captured != "tenant-ctx-check" {
		t.Errorf("context tenantID: got %q, want %q", captured, "tenant-ctx-check")
	}
}

// Regression: empty string query param must not be treated as valid tenant ID.
func TestTenantContext_EmptyQueryParam_Returns400(t *testing.T) {
	next := &tenantCapture{}
	handler := middleware.TenantContext(next)

	req := httptest.NewRequest(http.MethodGet, "/resource?tenantId=", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("empty query param: got %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// Regression: empty X-Tenant-ID header must not be treated as valid tenant ID.
func TestTenantContext_EmptyHeader_Returns400(t *testing.T) {
	next := &tenantCapture{}
	handler := middleware.TenantContext(next)

	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	req.Header.Set("X-Tenant-ID", "")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("empty X-Tenant-ID: got %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// setPathValue populates an http.Request path value (Go 1.22+).
// In tests we cannot go through a real ServeMux, so we use the stdlib API directly.
func setPathValue(r *http.Request, key, value string) {
	r.SetPathValue(key, value)
}