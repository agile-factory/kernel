package http

import (
	"encoding/json"
	"net/http"

	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/kernel/errors"
	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/kernel/response"
	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/auth/usecase"
	memberdomain "github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/membership/domain"
	memberusecase "github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/membership/usecase"
	"github.com/google/uuid"
)

// AuthHandler handles HTTP requests for authentication.
type AuthHandler struct {
	registerUC *usecase.RegisterUseCase
	loginUC    *usecase.LoginUseCase
	listUC     *memberusecase.ListMembersUseCase
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(
	registerUC *usecase.RegisterUseCase,
	loginUC *usecase.LoginUseCase,
	listUC *memberusecase.ListMembersUseCase,
) *AuthHandler {
	return &AuthHandler{
		registerUC: registerUC,
		loginUC:    loginUC,
		listUC:     listUC,
	}
}

// Register handles user registration.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, errors.BadRequest("Invalid request body"))
		return
	}

	if err := req.Validate(); err != nil {
		response.Error(w, errors.BadRequest(err.Error()))
		return
	}

	id := uuid.New().String()

	// Parse role, defaulting to ROLE_VIEWER for safety.
	// ROLE_OWNER is accepted here but should only be used by the tenant creation flow.
	role := memberdomain.MEMBER_ROLE_VIEWER
	switch memberdomain.MemberRole(req.Role) {
	case memberdomain.MEMBER_ROLE_OWNER:
		role = memberdomain.MEMBER_ROLE_OWNER
	case memberdomain.MEMBER_ROLE_ADMIN:
		role = memberdomain.MEMBER_ROLE_ADMIN
	case memberdomain.MEMBER_ROLE_AGENT:
		role = memberdomain.MEMBER_ROLE_AGENT
	}

	user, err := h.registerUC.Execute(r.Context(), id, req.TenantID, req.Email, req.Password, role)
	if err != nil {
		response.Error(w, err)
		return
	}

	// Fetch the memberships created during registration
	memberships, _ := h.listUC.ByUser(r.Context(), user.ID)
	userDTO := UserDTO{
		ID:          user.ID,
		Email:       user.Email.String(),
		Role:        string(user.Role),
		Memberships: toMembershipSummaries(memberships),
	}

	response.JSON(w, http.StatusCreated, userDTO)
}

// Login handles user login.
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

	token, user, err := h.loginUC.Execute(r.Context(), req.Email, req.Password)
	if err != nil {
		response.Error(w, err)
		return
	}

	// Fetch the user's current memberships for the response
	memberships, _ := h.listUC.ByUser(r.Context(), user.ID)

	resp := AuthResponse{
		Token: token,
		User: UserDTO{
			ID:          user.ID,
			Email:       user.Email.String(),
			Role:        string(user.Role),
			Memberships: toMembershipSummaries(memberships),
		},
	}

	response.JSON(w, http.StatusOK, resp)
}

// toMembershipSummaries converts domain memberships to compact DTO slices.
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
