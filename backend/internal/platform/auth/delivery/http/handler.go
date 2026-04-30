package http

import (
	"encoding/json"
	"net/http"

	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/kernel/errors"
	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/kernel/response"
	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/auth/usecase"
)

// AuthHandler handles HTTP requests for authentication.
type AuthHandler struct {
	registerUC *usecase.RegisterUseCase
	loginUC    *usecase.LoginUseCase
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(registerUC *usecase.RegisterUseCase, loginUC *usecase.LoginUseCase) *AuthHandler {
	return &AuthHandler{
		registerUC: registerUC,
		loginUC:    loginUC,
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

	// For a real app, generate a secure UUID.
	id := "uuid-stub" // TODO: Add a real UUID generator

	user, err := h.registerUC.Execute(r.Context(), id, req.TenantID, req.Email, req.Password)
	if err != nil {
		response.Error(w, err)
		return
	}

	// We could return a token here if auto-login is desired,
	// but standard register often just returns the user details.
	userDTO := UserDTO{
		ID:       user.ID,
		Email:    user.Email.String(),
		Role:     string(user.Role),
		TenantID: user.TenantID,
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

	resp := AuthResponse{
		Token: token,
		User: UserDTO{
			ID:       user.ID,
			Email:    user.Email.String(),
			Role:     string(user.Role),
			TenantID: user.TenantID,
		},
	}

	response.JSON(w, http.StatusOK, resp)
}
