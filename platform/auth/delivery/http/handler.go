package http

import (
	"encoding/json"
	"net/http"

	"github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/kernel/response"
	"github.com/agile-factory/kernel/platform/auth/domain"
	"github.com/agile-factory/kernel/platform/auth/usecase"
	memberdomain "github.com/agile-factory/kernel/platform/membership/domain"
	memberusecase "github.com/agile-factory/kernel/platform/membership/usecase"
)

// AuthHandler wires the auth use cases to HTTP endpoints.
type AuthHandler struct {
	signupOwnerUC  *usecase.SignupOwnerUseCase
	acceptInviteUC *usecase.AcceptInviteUseCase
	verifyEmailUC  *usecase.VerifyEmailUseCase
	loginUC        *usecase.LoginUseCase
	refreshUC      *usecase.RefreshTokenUseCase
	logoutUC       *usecase.LogoutUseCase
	listMembersUC  *memberusecase.ListMembersUseCase
}

// NewAuthHandler wires every use case the auth surface exposes.
func NewAuthHandler(
	signupOwnerUC *usecase.SignupOwnerUseCase,
	acceptInviteUC *usecase.AcceptInviteUseCase,
	verifyEmailUC *usecase.VerifyEmailUseCase,
	loginUC *usecase.LoginUseCase,
	refreshUC *usecase.RefreshTokenUseCase,
	logoutUC *usecase.LogoutUseCase,
	listMembersUC *memberusecase.ListMembersUseCase,
) *AuthHandler {
	return &AuthHandler{
		signupOwnerUC:  signupOwnerUC,
		acceptInviteUC: acceptInviteUC,
		verifyEmailUC:  verifyEmailUC,
		loginUC:        loginUC,
		refreshUC:      refreshUC,
		logoutUC:       logoutUC,
		listMembersUC:  listMembersUC,
	}
}

// SignupOwner handles POST /api/v1/auth/signup-owner.
func (h *AuthHandler) SignupOwner(w http.ResponseWriter, r *http.Request) {
	var req SignupOwnerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, errors.BadRequest("Invalid request body"))
		return
	}
	if err := req.Validate(); err != nil {
		response.Error(w, errors.BadRequest(err.Error()))
		return
	}

	out, err := h.signupOwnerUC.Execute(r.Context(), req.TenantName, req.TenantSlug, req.Email, req.Password)
	if err != nil {
		response.Error(w, err)
		return
	}

	memberships, _ := h.listMembersUC.ByUser(r.Context(), out.User.ID)
	response.JSON(w, http.StatusCreated, SignupOwnerResponse{
		TenantID:                 out.Tenant.ID,
		UserID:                   out.User.ID,
		MembershipID:             out.OwnerMembership.ID,
		AccessToken:              out.AccessToken,
		RefreshToken:             out.RefreshToken,
		EmailVerificationPending: !out.User.EmailVerified,
		User:                     toUserDTO(out.User, memberships),
	})
}

// AcceptInvite handles POST /api/v1/auth/accept-invite.
func (h *AuthHandler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	var req AcceptInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, errors.BadRequest("Invalid request body"))
		return
	}
	if err := req.Validate(); err != nil {
		response.Error(w, errors.BadRequest(err.Error()))
		return
	}

	out, err := h.acceptInviteUC.Execute(r.Context(), req.Token, req.Password)
	if err != nil {
		response.Error(w, err)
		return
	}

	memberships, _ := h.listMembersUC.ByUser(r.Context(), out.User.ID)
	response.JSON(w, http.StatusOK, AcceptInviteResponse{
		UserID:       out.User.ID,
		MembershipID: out.Membership.ID,
		TenantID:     out.Membership.TenantID,
		Role:         string(out.Membership.Role),
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
		User:         toUserDTO(out.User, memberships),
	})
}

// VerifyEmail handles POST /api/v1/auth/verify-email.
func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req VerifyEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, errors.BadRequest("Invalid request body"))
		return
	}
	if err := req.Validate(); err != nil {
		response.Error(w, errors.BadRequest(err.Error()))
		return
	}
	if err := h.verifyEmailUC.Execute(r.Context(), req.Token); err != nil {
		response.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Login handles POST /api/v1/auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, errors.BadRequest("Invalid request body"))
		return
	}
	if err := req.Validate(); err != nil {
		response.Error(w, errors.BadRequest(err.Error()))
		return
	}

	out, err := h.loginUC.Execute(r.Context(), req.Email, req.Password)
	if err != nil {
		response.Error(w, err)
		return
	}

	memberships, _ := h.listMembersUC.ByUser(r.Context(), out.User.ID)
	response.JSON(w, http.StatusOK, LoginResponse{
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
		User:         toUserDTO(out.User, memberships),
	})
}

// Refresh handles POST /api/v1/auth/refresh.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, errors.BadRequest("Invalid request body"))
		return
	}
	if err := req.Validate(); err != nil {
		response.Error(w, errors.BadRequest(err.Error()))
		return
	}

	out, err := h.refreshUC.Execute(r.Context(), req.RefreshToken)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, RefreshResponse{
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
	})
}

// Logout handles POST /api/v1/auth/logout. Idempotent.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req LogoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Empty body is allowed — treat as a no-op.
		req = LogoutRequest{}
	}
	if err := h.logoutUC.Execute(r.Context(), req.RefreshToken); err != nil {
		response.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// toUserDTO converts the domain User + the user's memberships into the wire
// shape returned by the auth endpoints.
func toUserDTO(user *domain.User, memberships []*memberdomain.Membership) UserDTO {
	return UserDTO{
		ID:            user.ID,
		Email:         user.Email.String(),
		Role:          string(user.Role),
		EmailVerified: user.EmailVerified,
		Status:        string(user.Status),
		Memberships:   toMembershipSummaries(memberships),
	}
}

func toMembershipSummaries(memberships []*memberdomain.Membership) []MembershipSummary {
	result := make([]MembershipSummary, len(memberships))
	for i, m := range memberships {
		result[i] = MembershipSummary{
			MembershipID: m.ID,
			TenantID:     m.TenantID,
			Role:         string(m.Role),
			Status:       string(m.Status),
		}
	}
	return result
}
