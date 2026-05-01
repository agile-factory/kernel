package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/kernel/errors"
	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/kernel/response"
	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/membership/domain"
	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/membership/usecase"
)

// MembershipHandler handles HTTP requests for membership management.
type MembershipHandler struct {
	inviteUC  *usecase.InviteMemberUseCase
	acceptUC  *usecase.AcceptInviteUseCase
	revokeUC  *usecase.RevokeMemberUseCase
	listUC    *usecase.ListMembersUseCase
}

// NewMembershipHandler creates a new MembershipHandler.
func NewMembershipHandler(
	inviteUC *usecase.InviteMemberUseCase,
	acceptUC *usecase.AcceptInviteUseCase,
	revokeUC *usecase.RevokeMemberUseCase,
	listUC *usecase.ListMembersUseCase,
) *MembershipHandler {
	return &MembershipHandler{
		inviteUC: inviteUC,
		acceptUC: acceptUC,
		revokeUC: revokeUC,
		listUC:   listUC,
	}
}

// Invite handles POST /memberships/invite
// The acting user ID is extracted from the X-User-ID header (set by auth middleware).
func (h *MembershipHandler) Invite(w http.ResponseWriter, r *http.Request) {
	inviterID := r.Header.Get("X-User-ID")
	if inviterID == "" {
		response.Error(w, errors.Unauthorized("authentication required"))
		return
	}

	var req InviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, errors.BadRequest("invalid request body"))
		return
	}
	if err := req.Validate(); err != nil {
		response.Error(w, errors.BadRequest(err.Error()))
		return
	}

	membership, err := h.inviteUC.Execute(r.Context(), inviterID, req.TenantID, req.InviteeUserID, domain.MemberRole(req.Role))
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, toMembershipResponse(membership))
}

// Accept handles POST /memberships/{id}/accept
func (h *MembershipHandler) Accept(w http.ResponseWriter, r *http.Request) {
	callerID := r.Header.Get("X-User-ID")
	if callerID == "" {
		response.Error(w, errors.Unauthorized("authentication required"))
		return
	}

	membershipID := r.PathValue("id")
	if membershipID == "" {
		response.Error(w, errors.BadRequest("membership id is required"))
		return
	}

	membership, err := h.acceptUC.Execute(r.Context(), membershipID, callerID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toMembershipResponse(membership))
}

// Revoke handles POST /memberships/{id}/revoke
func (h *MembershipHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	revokerID := r.Header.Get("X-User-ID")
	if revokerID == "" {
		response.Error(w, errors.Unauthorized("authentication required"))
		return
	}

	membershipID := r.PathValue("id")
	if membershipID == "" {
		response.Error(w, errors.BadRequest("membership id is required"))
		return
	}

	membership, err := h.revokeUC.Execute(r.Context(), membershipID, revokerID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toMembershipResponse(membership))
}

// ListByTenant handles GET /memberships/tenant/{tenantId}
// Supports pagination via ?limit=N&offset=N query params.
func (h *MembershipHandler) ListByTenant(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("tenantId")
	if tenantID == "" {
		response.Error(w, errors.BadRequest("tenantId is required"))
		return
	}

	memberships, err := h.listUC.ByTenant(r.Context(), tenantID)
	if err != nil {
		response.Error(w, err)
		return
	}

	limit, offset := parseMembershipPage(r)
	page := applyPage(memberships, limit, offset)

	result := make([]MembershipResponse, len(page))
	for i, m := range page {
		result[i] = toMembershipResponse(m)
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"data":   result,
		"total":  len(memberships),
		"limit":  limit,
		"offset": offset,
	})
}

// ListByUser handles GET /memberships/user/{userId}
// Supports pagination via ?limit=N&offset=N query params.
func (h *MembershipHandler) ListByUser(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("userId")
	if userID == "" {
		response.Error(w, errors.BadRequest("userId is required"))
		return
	}

	memberships, err := h.listUC.ByUser(r.Context(), userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	limit, offset := parseMembershipPage(r)
	page := applyPage(memberships, limit, offset)

	result := make([]MembershipResponse, len(page))
	for i, m := range page {
		result[i] = toMembershipResponse(m)
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"data":   result,
		"total":  len(memberships),
		"limit":  limit,
		"offset": offset,
	})
}

// parseMembershipPage reads limit and offset from query params.
// Defaults: limit=50, offset=0; max limit: 200.
func parseMembershipPage(r *http.Request) (limit, offset int) {
	limit, offset = 50, 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 200 {
		limit = 200
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 {
			offset = n
		}
	}
	return
}

// applyPage slices a membership list for in-memory pagination.
func applyPage(all []*domain.Membership, limit, offset int) []*domain.Membership {
	if offset >= len(all) {
		return nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end]
}
