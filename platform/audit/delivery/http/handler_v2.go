package http

import (
	"errors"
	"net/http"

	kernelauth "github.com/AgileFactory/kernel/kernel/auth"
	kernelerrors "github.com/AgileFactory/kernel/kernel/errors"
	"github.com/AgileFactory/kernel/kernel/response"
	auditdomain "github.com/AgileFactory/kernel/platform/audit/domain"
	"github.com/AgileFactory/kernel/platform/audit/usecase"
	memberdomain "github.com/AgileFactory/kernel/platform/membership/domain"
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
	// Read the caller from the context (populated by the Auth middleware after
	// JWT verification). Reading X-User-ID directly would be unsafe.
	callerID := kernelauth.UserIDFromContext(r.Context())
	if callerID == "" {
		response.Error(w, kernelerrors.Unauthorized("authentication required"))
		return
	}

	tenantID := r.PathValue("tenantId")
	if tenantID == "" {
		response.Error(w, kernelerrors.BadRequest("tenantId is required"))
		return
	}

	callerMembership, err := h.membershipRepo.FindByUserAndTenant(r.Context(), callerID, tenantID)
	if err != nil {
		// 403 only for an actual missing membership; bubble up real infra
		// failures so the kernel error layer logs them and returns 500.
		if errors.Is(err, memberdomain.ErrMembershipNotFound) {
			response.Error(w, kernelerrors.Forbidden("you do not have access to this tenant's audit logs"))
			return
		}
		response.Error(w, kernelerrors.WrapInternal(err))
		return
	}
	if !callerMembership.IsActive() || !callerMembership.IsAdmin() {
		response.Error(w, kernelerrors.Forbidden("only active admins can view audit logs"))
		return
	}

	// ── Cursor params ─────────────────────────────────────────────────────────
	limit := parseLimitV2(r)
	afterCursor := r.URL.Query().Get("after")

	// ── Fetch ─────────────────────────────────────────────────────────────────
	page, err := h.listUC.ByTenantCursor(r.Context(), tenantID, limit, afterCursor)
	if err != nil {
		writeCursorError(w, err)
		return
	}

	writeCursorPage(w, page, limit)
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
	if !kernelauth.IsPlatformAdmin(r.Context()) {
		response.Error(w, kernelerrors.Forbidden("only platform administrators can view global audit logs"))
		return
	}

	limit := parseLimitV2(r)
	afterCursor := r.URL.Query().Get("after")

	page, err := h.listUC.AllCursor(r.Context(), limit, afterCursor)
	if err != nil {
		writeCursorError(w, err)
		return
	}

	writeCursorPage(w, page, limit)
}

// writeCursorError maps a use-case error to the right HTTP status:
//   - ErrInvalidCursor → 400 with a generic message (no internal-error leakage)
//   - any other error  → 500 via WrapInternal (the kernel error layer logs it)
func writeCursorError(w http.ResponseWriter, err error) {
	if errors.Is(err, auditdomain.ErrInvalidCursor) {
		response.Error(w, kernelerrors.BadRequest("invalid cursor"))
		return
	}
	response.Error(w, kernelerrors.WrapInternal(err))
}

// writeCursorPage serialises a CursorPage into the v2 response envelope.
func writeCursorPage(w http.ResponseWriter, page *usecase.CursorPage, limit int) {
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
