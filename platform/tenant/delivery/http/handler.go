package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/kernel/errors"
	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/kernel/response"
	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/tenant/domain"
	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/tenant/usecase"
	"github.com/google/uuid"
)

// TenantHandler handles HTTP requests for tenant management.
type TenantHandler struct {
	createUC *usecase.CreateTenantUseCase
	findUC   *usecase.FindTenantUseCase
}

// NewTenantHandler creates a new TenantHandler.
func NewTenantHandler(createUC *usecase.CreateTenantUseCase, findUC *usecase.FindTenantUseCase) *TenantHandler {
	return &TenantHandler{
		createUC: createUC,
		findUC:   findUC,
	}
}

// Create handles POST /tenants — creates a new tenant.
func (h *TenantHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, errors.BadRequest("invalid request body"))
		return
	}

	if err := req.Validate(); err != nil {
		response.Error(w, errors.BadRequest(err.Error()))
		return
	}

	id := uuid.New().String()

	tenant, err := h.createUC.Execute(r.Context(), id, req.Name, req.Slug)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, toTenantResponse(tenant))
}

// GetByID handles GET /tenants/{id} — retrieves a tenant by UUID.
func (h *TenantHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		response.Error(w, errors.BadRequest("tenant id is required"))
		return
	}

	tenant, err := h.findUC.ByID(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toTenantResponse(tenant))
}

// GetBySlug handles GET /tenants/slug/{slug} — retrieves a tenant by slug.
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

	response.JSON(w, http.StatusOK, toTenantResponse(tenant))
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
