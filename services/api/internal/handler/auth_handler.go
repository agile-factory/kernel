package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/yourusername/immobilier-saas/services/api/internal/domain"
	"github.com/yourusername/immobilier-saas/services/api/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Register godoc
// @Summary      Register a new user
// @Description  Register a new user with email and password
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body domain.RegisterRequest true "Register request"
// @Success      201 {object} domain.User
// @Failure      400 {object} map[string]string
// @Failure      409 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req domain.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	user, err := h.authService.Register(c.Context(), req)
	if err != nil {
		// Check for specific errors
		if err.Error() == "user with this email already exists" {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "Email already registered",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to register user",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(user)
}

// Login godoc
// @Summary      Login user
// @Description  Authenticate user and return JWT tokens
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body domain.LoginRequest true "Login request"
// @Success      200 {object} domain.LoginResponse
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Failure      500 {object} map[string]string
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req domain.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Get device info and IP
	deviceInfo := c.Get("User-Agent")
	ipAddress := c.IP()

	response, err := h.authService.Login(c.Context(), req, deviceInfo, ipAddress)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid credentials",
		})
	}

	// Set refresh token as HTTP-only cookie
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    response.RefreshToken,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
		MaxAge:   604800, // 7 days
	})

	return c.JSON(fiber.Map{
		"access_token": response.AccessToken,
		"user":         response.User,
	})
}

// Logout godoc
// @Summary      Logout user
// @Description  Invalidate refresh token and clear cookie
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	refreshToken := c.Cookies("refresh_token")

	// Revoke refresh token if present
	if refreshToken != "" {
		if err := h.authService.Logout(c.Context(), refreshToken); err != nil {
			// Log error but don't fail
		}
	}

	// Clear refresh token cookie
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    "",
		HTTPOnly: true,
		MaxAge:   -1,
	})

	return c.JSON(fiber.Map{
		"message": "Logged out successfully",
	})
}

// RefreshToken godoc
// @Summary      Refresh access token
// @Description  Get new access token using refresh token from cookie
// @Tags         auth
// @Accept       json
// @Produce      json
// @Success      200 {object} map[string]string
// @Failure      401 {object} map[string]string
// @Router       /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *fiber.Ctx) error {
	refreshToken := c.Cookies("refresh_token")
	if refreshToken == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "No refresh token provided",
		})
	}

	response, err := h.authService.RefreshToken(c.Context(), refreshToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid refresh token",
		})
	}

	return c.JSON(fiber.Map{
		"access_token": response.AccessToken,
		"user":         response.User,
	})
}

// ForgotPassword godoc
// @Summary      Request password reset
// @Description  Send password reset email to user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body domain.ForgotPasswordRequest true "Forgot password request"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Router       /auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *fiber.Ctx) error {
	var req domain.ForgotPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	err := h.authService.ForgotPassword(c.Context(), req.Email)
	if err != nil {
		// Don't reveal if email exists
		return c.JSON(fiber.Map{
			"message": "If the email exists, a password reset link has been sent",
		})
	}

	return c.JSON(fiber.Map{
		"message": "If the email exists, a password reset link has been sent",
	})
}

// ResetPassword godoc
// @Summary      Reset password
// @Description  Reset user password using token from email
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body domain.ResetPasswordRequest true "Reset password request"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Router       /auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *fiber.Ctx) error {
	var req domain.ResetPasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	err := h.authService.ResetPassword(c.Context(), req.Token, req.Password)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid or expired token",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Password reset successfully",
	})
}

// VerifyEmail godoc
// @Summary      Verify email
// @Description  Verify user email using token from email
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        token path string true "Verification token"
// @Success      200 {object} map[string]string
// @Failure      400 {object} map[string]string
// @Router       /auth/verify-email/{token} [get]
func (h *AuthHandler) VerifyEmail(c *fiber.Ctx) error {
	token := c.Params("token")
	if token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Token is required",
		})
	}

	err := h.authService.VerifyEmail(c.Context(), token)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid or expired token",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Email verified successfully",
	})
}
