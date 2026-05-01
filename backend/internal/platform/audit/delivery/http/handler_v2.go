package http

import (
	"net/http"

	kernelauth "github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/kernel/auth"
	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/kernel/errors"
	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/kernel/response"
	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/audit/usecase"
	memberdomain "github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/membership/domain"
)

// AuditHandlerV2 serves the cursor-based audit-log endpoints under /api/v2/.
//
// It shares the same use case and membership repository as AuditHandler (v1)
// so auth guards are identical — only the pagination contract changes.
type AuditHandlerV2 struct {
	listUC         *usecase.ListAuditLogsUseCase
	membershipRepo memberdomain.MembershipRepository
}

// NewAuditHandlerV2 creates a new AuditHandlerV2.
func NewAuditHandlerV2(listUC *usecase.ListAuditLogsUseCase, membershipRepo memberdomain.MembershipRepository) *AuditHandlerV2 {
	return &AuditHandlerV2{
		listUC:         listUC,
		membershipRepo: membershipRepo,
	}
}

// ListByTenantV2 handles GET /api/v2/tenants/{tenantId}/audit-logs
//
// Query params:
//
//	limit — number of results per page (default 50, max 200)
//	after — opaque cursor token from a previous response's next_cursor field
//
// Requires the calling user to be an active ROLE_ADMIN or ROLE_OWNER of the tenant.
func (h *AuditHandlerV2) ListByTenantV2(w http.ResponseWriter, r *http.Request) {
	// ── Auth guard ────────────────────────────────────────────────────────────
	callerID := r.Header.Get("X-User-ID")
	if callerID == "" {
		response.Error(w, errors.Unauthorized("authentication required"))
		return
	}

	tenantID := r.PathValue("tenantId")
	if tenantID == "" {
		response.Error(w, errors.BadRequest("tenantId is required"))
		return
	}

	callerMembership, err := h.membershipRepo.FindByUserAndTenant(r.Context(), callerID, tenantID)
	if err != nil {
		response.Error(w, errors.Forbidden("you do not have access to this tenant's audit logs"))
		return
	}
	if !callerMembership.IsActive() || !callerMembership.IsAdmin() {
		response.Error(w, errors.Forbidden("only active admins can view audit logs"))
		return
	}

	// ── Cursor params ─────────────────────────────────────────────────────────
	limit := parseLimitV2(r)
	afterCursor := r.URL.Query().Get("after")

	// ── Fetch ─────────────────────────────────────────────────────────────────
	page, err := h.listUC.ByTenantCursor(r.Context(), tenantID, limit, afterCursor)
	if err != nil {
		// Distinguish a malformed cursor (client error) from an internal error.
		response.Error(w, errors.BadRequest("invalid cursor: "+err.Error()))
		return
	}

	// ── Serialize ─────────────────────────────────────────────────────────────
	data := make([]AuditLogResponse, len(page.Logs))
	for i, l := range page.Logs {
		data[i] = toAuditLogResponse(l)
	}

	response.JSON(w, http.StatusOK, AuditLogListResponseV2{
		Data:       data,
		Total:      -1,
		Limit:      limit,
		Offset:     0,
		NextCursor: page.NextCursor,
		HasMore:    page.HasMore,
	})
}

// ListAllV2 handles GET /api/v2/audit-logs
//
// Query params:
//
//	limit — number of results per page (default 50, max 200)
//	after — opaque cursor token from a previous response's next_cursor field
//
// Requires the calling user to have the platform-level ROLE_ADMIN.
func (h *AuditHandlerV2) ListAllV2(w http.ResponseWriter, r *http.Request) {
	// ── Auth guard ────────────────────────────────────────────────────────────
	if !kernelauth.IsPlatformAdmin(r.Context()) {
		response.Error(w, errors.Forbidden("only platform administrators can view global audit logs"))
		return
	}

	// ── Cursor params ─────────────────────────────────────────────────────────
	limit := parseLimitV2(r)
	afterCursor := r.URL.Query().Get("after")

	// ── Fetch ─────────────────────────────────────────────────────────────────
	page, err := h.listUC.AllCursor(r.Context(), limit, afterCursor)
	if err != nil {
		response.Error(w, errors.BadRequest("invalid cursor: "+err.Error()))
		return
	}

	// ── Serialize ─────────────────────────────────────────────────────────────
	data := make([]AuditLogResponse, len(page.Logs))
	for i, l := range page.Logs {
		data[i] = toAuditLogResponse(l)
	}

	response.JSON(w, http.StatusOK, AuditLogListResponseV2{
		Data:       data,
		Total:      -1,
		Limit:      limit,
		Offset:     0,
		NextCursor: page.NextCursor,
		HasMore:    page.HasMore,
	})
}

// parseLimitV2 extracts limit from query params (same rules as v1, no offset).
func parseLimitV2(r *http.Request) int {
	limit, _ := parsePagination(r) // reuse v1 helper; offset is ignored
	return limit
}
