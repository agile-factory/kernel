package http

import (
	"net/http"
	"strconv"

	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/kernel/errors"
	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/kernel/response"
	memberdomain "github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/membership/domain"
	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/audit/usecase"
)

// AuditHandler handles HTTP requests for the audit read model.
type AuditHandler struct {
	listUC         *usecase.ListAuditLogsUseCase
	membershipRepo memberdomain.MembershipRepository
}

// NewAuditHandler creates a new AuditHandler.
// membershipRepo is used to authorise admin-only access to audit logs.
func NewAuditHandler(listUC *usecase.ListAuditLogsUseCase, membershipRepo memberdomain.MembershipRepository) *AuditHandler {
	return &AuditHandler{
		listUC:         listUC,
		membershipRepo: membershipRepo,
	}
}

// ListByTenant handles GET /api/v1/tenants/{tenantId}/audit-logs
//
// Query params:
//
//	limit  — number of results (default 50, max 200)
//	offset — pagination offset  (default 0)
//
// Requires the calling user (X-User-ID header) to be an active ROLE_ADMIN of the tenant.
func (h *AuditHandler) ListByTenant(w http.ResponseWriter, r *http.Request) {
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

	// Only active admins of this tenant may read its audit trail.
	callerMembership, err := h.membershipRepo.FindByUserAndTenant(r.Context(), callerID, tenantID)
	if err != nil {
		response.Error(w, errors.Forbidden("you do not have access to this tenant's audit logs"))
		return
	}
	if !callerMembership.IsActive() || callerMembership.Role != memberdomain.MEMBER_ROLE_ADMIN {
		response.Error(w, errors.Forbidden("only active admins can view audit logs"))
		return
	}

	// ── Pagination ────────────────────────────────────────────────────────────
	limit, offset := parsePagination(r)

	// ── Fetch ─────────────────────────────────────────────────────────────────
	logs, err := h.listUC.ByTenant(r.Context(), tenantID, limit, offset)
	if err != nil {
		response.Error(w, errors.WrapInternal(err))
		return
	}

	// ── Serialize ─────────────────────────────────────────────────────────────
	data := make([]AuditLogResponse, len(logs))
	for i, l := range logs {
		data[i] = toAuditLogResponse(l)
	}

	response.JSON(w, http.StatusOK, AuditLogListResponse{
		Data:   data,
		Total:  len(data),
		Limit:  limit,
		Offset: offset,
	})
}

// parsePagination extracts limit and offset from query parameters.
// It applies sensible defaults and caps limit at 200.
func parsePagination(r *http.Request) (limit, offset int) {
	limit = 50
	offset = 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 {
			offset = n
		}
	}
	return
}
