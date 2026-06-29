// Package http exposes the end_customers platform's HTTP surface.
package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	kerrors "github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/kernel/response"
	"github.com/agile-factory/kernel/platform/end_customers/domain"
	"github.com/agile-factory/kernel/platform/end_customers/usecase"
)

// EndCustomerHandler handles HTTP requests for the end_customers surface.
type EndCustomerHandler struct {
	createUC  *usecase.CreateEndCustomerUseCase
	getUC     *usecase.GetEndCustomerUseCase
	updateUC  *usecase.UpdateEndCustomerUseCase
	deleteUC  *usecase.DeleteEndCustomerUseCase
	suspendUC *usecase.SuspendEndCustomerUseCase
	restoreUC *usecase.RestoreEndCustomerUseCase
	listUC    *usecase.ListEndCustomersUseCase
}

// NewEndCustomerHandler wires the handler with all use cases.
func NewEndCustomerHandler(
	createUC *usecase.CreateEndCustomerUseCase,
	getUC *usecase.GetEndCustomerUseCase,
	updateUC *usecase.UpdateEndCustomerUseCase,
	deleteUC *usecase.DeleteEndCustomerUseCase,
	suspendUC *usecase.SuspendEndCustomerUseCase,
	restoreUC *usecase.RestoreEndCustomerUseCase,
	listUC *usecase.ListEndCustomersUseCase,
) *EndCustomerHandler {
	return &EndCustomerHandler{
		createUC:  createUC,
		getUC:     getUC,
		updateUC:  updateUC,
		deleteUC:  deleteUC,
		suspendUC: suspendUC,
		restoreUC: restoreUC,
		listUC:    listUC,
	}
}

// ── DTOs ──────────────────────────────────────────────────────────────────────

type endCustomerResponse struct {
	ID         string            `json:"id"`
	TenantID   string            `json:"tenantId"`
	ExternalID string            `json:"externalId,omitempty"`
	Email      string            `json:"email,omitempty"`
	Phone      string            `json:"phone,omitempty"`
	FirstName  string            `json:"firstName"`
	LastName   string            `json:"lastName"`
	FullName   string            `json:"fullName"`
	Status     string            `json:"status"`
	Tags       []string          `json:"tags"`
	Metadata   map[string]string `json:"metadata"`
	CreatedAt  string            `json:"createdAt"`
	UpdatedAt  string            `json:"updatedAt"`
}

func toEndCustomerResponse(ec *domain.EndCustomer) endCustomerResponse {
	tags := ec.Tags
	if tags == nil {
		tags = []string{}
	}
	metadata := ec.Metadata
	if metadata == nil {
		metadata = map[string]string{}
	}
	return endCustomerResponse{
		ID:         ec.ID,
		TenantID:   ec.TenantID,
		ExternalID: ec.ExternalID,
		Email:      ec.Email,
		Phone:      ec.Phone,
		FirstName:  ec.FirstName,
		LastName:   ec.LastName,
		FullName:   ec.FullName(),
		Status:     string(ec.Status),
		Tags:       tags,
		Metadata:   metadata,
		CreatedAt:  ec.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  ec.UpdatedAt.Format(time.RFC3339),
	}
}

// ── Create ────────────────────────────────────────────────────────────────────

type createEndCustomerRequest struct {
	ExternalID string            `json:"externalId"`
	Email      string            `json:"email"`
	Phone      string            `json:"phone"`
	FirstName  string            `json:"firstName"`
	LastName   string            `json:"lastName"`
	Tags       []string          `json:"tags"`
	Metadata   map[string]string `json:"metadata"`
}

// Create handles POST /api/v1/end-customers
func (h *EndCustomerHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body createEndCustomerRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, kerrors.BadRequest("invalid JSON body"))
		return
	}

	out, err := h.createUC.Execute(r.Context(), usecase.CreateEndCustomerInput{
		ExternalID: body.ExternalID,
		Email:      body.Email,
		Phone:      body.Phone,
		FirstName:  body.FirstName,
		LastName:   body.LastName,
		Tags:       body.Tags,
		Metadata:   body.Metadata,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, toEndCustomerResponse(out.EndCustomer))
}

// ── Get ───────────────────────────────────────────────────────────────────────

// Get handles GET /api/v1/end-customers/{id}
func (h *EndCustomerHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		response.Error(w, kerrors.BadRequest("id path parameter is required"))
		return
	}

	out, err := h.getUC.Execute(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toEndCustomerResponse(out.EndCustomer))
}

// ── Update ────────────────────────────────────────────────────────────────────

type patchEndCustomerRequest struct {
	ExternalID *string           `json:"externalId"`
	Email      *string           `json:"email"`
	Phone      *string           `json:"phone"`
	FirstName  *string           `json:"firstName"`
	LastName   *string           `json:"lastName"`
	Tags       []string          `json:"tags"`
	Metadata   map[string]string `json:"metadata"`
}

// Update handles PATCH /api/v1/end-customers/{id}
func (h *EndCustomerHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		response.Error(w, kerrors.BadRequest("id path parameter is required"))
		return
	}

	var body patchEndCustomerRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, kerrors.BadRequest("invalid JSON body"))
		return
	}

	// Tags and Metadata are "set" when the key appears in JSON.
	// We use the decoded body directly; nil slice/map = not provided.
	out, err := h.updateUC.Execute(r.Context(), usecase.UpdateEndCustomerInput{
		ID:          id,
		ExternalID:  body.ExternalID,
		Email:       body.Email,
		Phone:       body.Phone,
		FirstName:   body.FirstName,
		LastName:    body.LastName,
		Tags:        body.Tags,
		TagsSet:     body.Tags != nil,
		Metadata:    body.Metadata,
		MetadataSet: body.Metadata != nil,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toEndCustomerResponse(out.EndCustomer))
}

// ── Delete ────────────────────────────────────────────────────────────────────

// Delete handles DELETE /api/v1/end-customers/{id}
func (h *EndCustomerHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		response.Error(w, kerrors.BadRequest("id path parameter is required"))
		return
	}

	if err := h.deleteUC.Execute(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusNoContent, nil)
}

// ── Suspend ───────────────────────────────────────────────────────────────────

// Suspend handles POST /api/v1/end-customers/{id}/suspend
func (h *EndCustomerHandler) Suspend(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		response.Error(w, kerrors.BadRequest("id path parameter is required"))
		return
	}

	if err := h.suspendUC.Execute(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusNoContent, nil)
}

// ── Restore ───────────────────────────────────────────────────────────────────

// Restore handles POST /api/v1/end-customers/{id}/restore
func (h *EndCustomerHandler) Restore(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		response.Error(w, kerrors.BadRequest("id path parameter is required"))
		return
	}

	if err := h.restoreUC.Execute(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusNoContent, nil)
}

// ── List ──────────────────────────────────────────────────────────────────────

type listEndCustomersResponse struct {
	Items      []endCustomerResponse `json:"items"`
	NextCursor string                `json:"nextCursor,omitempty"`
}

// List handles GET /api/v1/end-customers
func (h *EndCustomerHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit := 20
	if l := q.Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	var statusFilter *domain.EndCustomerStatus
	if s := q.Get("status"); s != "" {
		st := domain.EndCustomerStatus(s)
		statusFilter = &st
	}

	out, err := h.listUC.Execute(r.Context(), usecase.ListEndCustomersInput{
		Cursor: q.Get("cursor"),
		Limit:  limit,
		Status: statusFilter,
		Tag:    q.Get("tag"),
		Search: q.Get("search"),
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	items := make([]endCustomerResponse, len(out.Items))
	for i, ec := range out.Items {
		items[i] = toEndCustomerResponse(ec)
	}

	response.JSON(w, http.StatusOK, listEndCustomersResponse{
		Items:      items,
		NextCursor: out.NextCursor,
	})
}
