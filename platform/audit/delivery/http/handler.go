package http

import (
	stderrors "errors"
	"net/http"
	"strconv"

	kernelauth "github.com/agile-factory/kernel/kernel/auth"
	"github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/kernel/response"
	"github.com/agile-factory/kernel/platform/audit/usecase"
	memberdomain "github.com/agile-factory/kernel/platform/membership/domain"
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
	// Read the caller from the request context (populated by the Auth middleware
	// after JWT verification). Reading from the X-User-ID header would be unsafe
	// because clients can send that header directly.
	callerID := kernelauth.UserIDFromContext(r.Context())
	if callerID == "" {
		response.Error(w, errors.Unauthorized("authentication required"))
		return
	}

	tenantID := r.PathValue("tenantId")
	if tenantID == "" {
		response.Error(w, errors.BadRequest("tenantId is required"))
		return
	}

	// Only active admins (ROLE_ADMIN or ROLE_OWNER) of this tenant may read its audit trail.
	callerMembership, err := h.membershipRepo.FindByUserAndTenant(r.Context(), callerID, tenantID)
	if err != nil {
		if stderrors.Is(err, memberdomain.ErrMembershipNotFound) {
			response.Error(w, errors.Forbidden("you do not have access to this tenant's audit logs"))
			return
		}
		response.Error(w, errors.WrapInternal(err))
		return
	}
	if !callerMembership.IsActive() || !callerMembership.IsAdmin() {
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

// ListAll handles GET /api/v1/audit-logs
//
// Query params:
//
//	limit  — number of results (default 50, max 200)
//	offset — pagination offset  (default 0)
//
// Requires the calling user to have the platform-level ROLE_ADMIN.
func (h *AuditHandler) ListAll(w http.ResponseWriter, r *http.Request) {
	// ── Auth guard ────────────────────────────────────────────────────────────
	// Only platform super-admins can view all audit logs across the entire system.
	// This relies on the JWT role injected by the Auth middleware into the context.
	if !kernelauth.IsPlatformAdmin(r.Context()) {
		response.Error(w, errors.Forbidden("only platform administrators can view global audit logs"))
		return
	}

	// ── Pagination ────────────────────────────────────────────────────────────
	limit, offset := parsePagination(r)

	// ── Fetch ─────────────────────────────────────────────────────────────────
	logs, err := h.listUC.All(r.Context(), limit, offset)
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
