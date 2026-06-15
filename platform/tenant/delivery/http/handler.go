package http

import (
	"encoding/json"
	stderrors "errors"
	"net/http"
	"time"

	kernelauth "github.com/agile-factory/kernel/kernel/auth"
	"github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/kernel/response"
	memberdomain "github.com/agile-factory/kernel/platform/membership/domain"
	"github.com/agile-factory/kernel/platform/tenant/domain"
	"github.com/agile-factory/kernel/platform/tenant/usecase"
	"github.com/google/uuid"
)

// TenantHandler wires the tenant use cases to HTTP endpoints. Membership
// lookups for permission gating use the injected MembershipRepo (an
// interface — kept narrow on purpose to avoid pulling membership/usecase).
type TenantHandler struct {
	createUC         *usecase.CreateTenantUseCase
	findUC           *usecase.FindTenantUseCase
	updateUC         *usecase.UpdateTenantUseCase
	deleteUC         *usecase.DeleteTenantUseCase
	getSettingsUC    *usecase.GetTenantSettingsUseCase
	updateSettingsUC *usecase.UpdateTenantSettingsUseCase
	memberships      memberdomain.MembershipRepository
}

// NewTenantHandler wires every dependency the tenant surface needs.
func NewTenantHandler(
	createUC *usecase.CreateTenantUseCase,
	findUC *usecase.FindTenantUseCase,
	updateUC *usecase.UpdateTenantUseCase,
	deleteUC *usecase.DeleteTenantUseCase,
	getSettingsUC *usecase.GetTenantSettingsUseCase,
	updateSettingsUC *usecase.UpdateTenantSettingsUseCase,
	memberships memberdomain.MembershipRepository,
) *TenantHandler {
	return &TenantHandler{
		createUC:         createUC,
		findUC:           findUC,
		updateUC:         updateUC,
		deleteUC:         deleteUC,
		getSettingsUC:    getSettingsUC,
		updateSettingsUC: updateSettingsUC,
		memberships:      memberships,
	}
}

// AdminCreate handles POST /admin/v1/tenants — super-admin only.
func (h *TenantHandler) AdminCreate(w http.ResponseWriter, r *http.Request) {
	if !kernelauth.IsPlatformAdmin(r.Context()) {
		response.Error(w, errors.Forbidden("only platform administrators can create tenants"))
		return
	}

	var req CreateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, errors.BadRequest("invalid request body"))
		return
	}
	if err := req.Validate(); err != nil {
		response.Error(w, errors.BadRequest(err.Error()))
		return
	}

	tenant, err := h.createUC.Execute(r.Context(), uuid.New().String(), req.Name, req.Slug)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, toTenantResponse(tenant))
}

// GetByID handles GET /api/v1/tenants/{id}.
func (h *TenantHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		response.Error(w, errors.BadRequest("tenant id is required"))
		return
	}
	if _, err := h.requireActiveMembership(r, id, false); err != nil {
		response.Error(w, err)
		return
	}

	tenant, err := h.findUC.ByID(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, toTenantResponse(tenant))
}

// GetBySlug handles GET /api/v1/tenants/slug/{slug}.
func (h *TenantHandler) GetBySlug(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		response.Error(w, errors.BadRequest("slug is required"))
		return
	}
	tenant, err := h.findUC.BySlug(r.Context(), slug)
	if err != nil {
		response.Error(w, err)
		return
	}
	if _, err := h.requireActiveMembership(r, tenant.ID, false); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, toTenantResponse(tenant))
}

// Update handles PATCH /api/v1/tenants/{id} — OWNER/ADMIN only.
func (h *TenantHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		response.Error(w, errors.BadRequest("tenant id is required"))
		return
	}
	callerID, err := h.requireActiveMembership(r, id, true)
	if err != nil {
		response.Error(w, err)
		return
	}

	var req UpdateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, errors.BadRequest("invalid request body"))
		return
	}
	if err := req.Validate(); err != nil {
		response.Error(w, errors.BadRequest(err.Error()))
		return
	}

	tenant, err := h.updateUC.Execute(r.Context(), id, callerID, req.Name)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, toTenantResponse(tenant))
}

// Delete handles DELETE /api/v1/tenants/{id} — OWNER only.
func (h *TenantHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		response.Error(w, errors.BadRequest("tenant id is required"))
		return
	}
	callerID, err := h.requireActiveOwner(r, id)
	if err != nil {
		response.Error(w, err)
		return
	}
	if err := h.deleteUC.Execute(r.Context(), id, callerID); err != nil {
		response.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GetSettings handles GET /api/v1/tenants/{id}/settings.
func (h *TenantHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		response.Error(w, errors.BadRequest("tenant id is required"))
		return
	}
	if _, err := h.requireActiveMembership(r, id, false); err != nil {
		response.Error(w, err)
		return
	}
	s, err := h.getSettingsUC.Execute(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, toSettingsResponse(s))
}

// UpdateSettings handles PATCH /api/v1/tenants/{id}/settings — OWNER/ADMIN only.
func (h *TenantHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		response.Error(w, errors.BadRequest("tenant id is required"))
		return
	}
	callerID, err := h.requireActiveMembership(r, id, true)
	if err != nil {
		response.Error(w, err)
		return
	}

	var req UpdateSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, errors.BadRequest("invalid request body"))
		return
	}

	s, err := h.updateSettingsUC.Execute(r.Context(), id, callerID, req.ToPatch())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, toSettingsResponse(s))
}

// requireActiveMembership returns the caller's user ID and an HTTP-ready
// error if the membership check fails. When adminOnly is true, only
// ROLE_OWNER and ROLE_ADMIN are accepted; platform admins bypass entirely.
func (h *TenantHandler) requireActiveMembership(r *http.Request, tenantID string, adminOnly bool) (string, error) {
	callerID := kernelauth.UserIDFromContext(r.Context())
	if callerID == "" {
		return "", errors.Unauthorized("authentication required")
	}
	if kernelauth.IsPlatformAdmin(r.Context()) {
		return callerID, nil
	}

	mship, err := h.memberships.FindByUserAndTenant(r.Context(), callerID, tenantID)
	if err != nil {
		if stderrors.Is(err, memberdomain.ErrMembershipNotFound) {
			return "", errors.Forbidden("you do not have access to this tenant")
		}
		return "", errors.WrapInternal(err)
	}
	if !mship.IsActive() {
		return "", errors.Forbidden("you do not have access to this tenant")
	}
	if adminOnly && !mship.IsAdmin() {
		return "", errors.Forbidden("only tenant admins can perform this action")
	}
	return callerID, nil
}

// requireActiveOwner is the tightest guard — only ROLE_OWNER (or platform admin).
func (h *TenantHandler) requireActiveOwner(r *http.Request, tenantID string) (string, error) {
	callerID := kernelauth.UserIDFromContext(r.Context())
	if callerID == "" {
		return "", errors.Unauthorized("authentication required")
	}
	if kernelauth.IsPlatformAdmin(r.Context()) {
		return callerID, nil
	}
	mship, err := h.memberships.FindByUserAndTenant(r.Context(), callerID, tenantID)
	if err != nil {
		if stderrors.Is(err, memberdomain.ErrMembershipNotFound) {
			return "", errors.Forbidden("only the tenant owner can perform this action")
		}
		return "", errors.WrapInternal(err)
	}
	if !mship.IsActive() || !mship.IsOwner() {
		return "", errors.Forbidden("only the tenant owner can perform this action")
	}
	return callerID, nil
}

// toTenantResponse converts a domain Tenant to the HTTP response DTO.
func toTenantResponse(t *domain.Tenant) TenantResponse {
	return TenantResponse{
		ID:        t.ID,
		Name:      t.Name,
		Slug:      t.Slug,
		IsActive:  t.IsActive,
		CreatedAt: t.CreatedAt.Format(time.RFC3339),
	}
}
