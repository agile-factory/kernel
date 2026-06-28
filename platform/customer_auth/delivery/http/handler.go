package http

import (
	"encoding/json"
	"net/http"

	kerrors "github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/kernel/response"
	"github.com/agile-factory/kernel/platform/customer_auth/usecase"
)

// CustomerAuthHandler handles HTTP requests for the customer_auth surface.
type CustomerAuthHandler struct {
	registerUC      *usecase.CustomerRegisterUseCase
	loginUC         *usecase.CustomerLoginUseCase
	refreshUC       *usecase.CustomerRefreshUseCase
	logoutUC        *usecase.CustomerLogoutUseCase
	forgotPasswordUC *usecase.CustomerForgotPasswordUseCase
	resetPasswordUC *usecase.CustomerResetPasswordUseCase
}

// NewCustomerAuthHandler wires the handler.
func NewCustomerAuthHandler(
	registerUC *usecase.CustomerRegisterUseCase,
	loginUC *usecase.CustomerLoginUseCase,
	refreshUC *usecase.CustomerRefreshUseCase,
	logoutUC *usecase.CustomerLogoutUseCase,
	forgotPasswordUC *usecase.CustomerForgotPasswordUseCase,
	resetPasswordUC *usecase.CustomerResetPasswordUseCase,
) *CustomerAuthHandler {
	return &CustomerAuthHandler{
		registerUC:       registerUC,
		loginUC:          loginUC,
		refreshUC:        refreshUC,
		logoutUC:         logoutUC,
		forgotPasswordUC: forgotPasswordUC,
		resetPasswordUC:  resetPasswordUC,
	}
}

// ── DTOs ─────────────────────────────────────────────────────────────────────

type registerRequest struct {
	EndCustomerID string `json:"end_customer_id"`
	Password      string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type tokenPairResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

type resetPasswordRequest struct {
	ResetToken  string `json:"reset_token"`
	NewPassword string `json:"new_password"`
}

// ── Handlers ─────────────────────────────────────────────────────────────────

// Register handles POST /api/v1/tenants/{id}/customers/auth/register
func (h *CustomerAuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("id")
	var body registerRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, kerrors.BadRequest("invalid JSON body"))
		return
	}
	if body.EndCustomerID == "" || body.Password == "" {
		response.Error(w, kerrors.BadRequest("end_customer_id and password are required"))
		return
	}
	if err := h.registerUC.Execute(r.Context(), usecase.CustomerRegisterInput{
		EndCustomerID: body.EndCustomerID,
		TenantID:      tenantID,
		Password:      body.Password,
	}); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, nil)
}

// Login handles POST /api/v1/tenants/{id}/customers/auth/login
func (h *CustomerAuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("id")
	var body loginRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, kerrors.BadRequest("invalid JSON body"))
		return
	}
	if body.Email == "" || body.Password == "" {
		response.Error(w, kerrors.BadRequest("email and password are required"))
		return
	}
	out, err := h.loginUC.Execute(r.Context(), usecase.CustomerLoginInput{
		Email:    body.Email,
		Password: body.Password,
		TenantID: tenantID,
	})
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, tokenPairResponse{
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
	})
}

// Refresh handles POST /api/v1/tenants/{id}/customers/auth/refresh
func (h *CustomerAuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var body refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, kerrors.BadRequest("invalid JSON body"))
		return
	}
	if body.RefreshToken == "" {
		response.Error(w, kerrors.BadRequest("refresh_token is required"))
		return
	}
	out, err := h.refreshUC.Execute(r.Context(), body.RefreshToken)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, tokenPairResponse{
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
	})
}

// Logout handles POST /api/v1/tenants/{id}/customers/auth/logout
func (h *CustomerAuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var body logoutRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, kerrors.BadRequest("invalid JSON body"))
		return
	}
	if err := h.logoutUC.Execute(r.Context(), body.RefreshToken); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusNoContent, nil)
}

// ForgotPassword handles POST /api/v1/tenants/{id}/customers/auth/forgot-password
func (h *CustomerAuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	tenantID := r.PathValue("id")
	var body forgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, kerrors.BadRequest("invalid JSON body"))
		return
	}
	if body.Email == "" {
		response.Error(w, kerrors.BadRequest("email is required"))
		return
	}
	if err := h.forgotPasswordUC.Execute(r.Context(), usecase.CustomerForgotPasswordInput{
		Email:    body.Email,
		TenantID: tenantID,
	}); err != nil {
		response.Error(w, err)
		return
	}
	// Always 204 — even when email unknown (enumeration resistance).
	response.JSON(w, http.StatusNoContent, nil)
}

// ResetPassword handles POST /api/v1/tenants/{id}/customers/auth/reset-password
func (h *CustomerAuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var body resetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, kerrors.BadRequest("invalid JSON body"))
		return
	}
	if body.ResetToken == "" || body.NewPassword == "" {
		response.Error(w, kerrors.BadRequest("reset_token and new_password are required"))
		return
	}
	if err := h.resetPasswordUC.Execute(r.Context(), usecase.CustomerResetPasswordInput{
		ResetToken:  body.ResetToken,
		NewPassword: body.NewPassword,
	}); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusNoContent, nil)
}

// writeErr is a fallback used by the middleware before the handler is reached.
func writeErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write([]byte(`{"error":"` + msg + `"}`))
}
