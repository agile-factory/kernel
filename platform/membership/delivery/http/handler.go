package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	kernelauth "github.com/AgileFactory/kernel/kernel/auth"
	"github.com/AgileFactory/kernel/kernel/errors"
	"github.com/AgileFactory/kernel/kernel/response"
	"github.com/AgileFactory/kernel/platform/membership/domain"
	"github.com/AgileFactory/kernel/platform/membership/usecase"
)

// MembershipHandler handles HTTP requests for membership management.
//
// Per RFC-0018 §6 the Accept flow no longer lives here — the invitee
// redeems an invite token via POST /api/v1/auth/accept-invite.
type MembershipHandler struct {
	inviteUC *usecase.InviteMemberUseCase
	revokeUC *usecase.RevokeMemberUseCase
	listUC   *usecase.ListMembersUseCase
}

func NewMembershipHandler(
	inviteUC *usecase.InviteMemberUseCase,
	revokeUC *usecase.RevokeMemberUseCase,
	listUC *usecase.ListMembersUseCase,
) *MembershipHandler {
	return &MembershipHandler{
		inviteUC: inviteUC,
		revokeUC: revokeUC,
		listUC:   listUC,
	}
}

// Invite handles POST /api/v1/memberships/invite. The acting user ID comes
// from the auth middleware context — never from a client header.
func (h *MembershipHandler) Invite(w http.ResponseWriter, r *http.Request) {
	inviterID := kernelauth.UserIDFromContext(r.Context())
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

	out, err := h.inviteUC.Execute(r.Context(), inviterID, req.TenantID, req.Email, domain.MemberRole(req.Role))
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, toInviteResponse(out.Invite, out.RawToken))
}

// Revoke handles POST /api/v1/memberships/{id}/revoke.
func (h *MembershipHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	revokerID := kernelauth.UserIDFromContext(r.Context())
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

// ListByTenant handles GET /api/v1/memberships/tenant/{tenantId}. Supports
// pagination via ?limit=N&offset=N (max limit 200).
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
	h.respondList(w, r, memberships)
}

// ListByUser handles GET /api/v1/memberships/user/{userId}.
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
	h.respondList(w, r, memberships)
}

func (h *MembershipHandler) respondList(w http.ResponseWriter, r *http.Request, memberships []*domain.Membership) {
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

func applyPage(all []*domain.Membership, limit, offset int) []*domain.Membership {
	if offset >= len(all) {
		return []*domain.Membership{}
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end]
}
