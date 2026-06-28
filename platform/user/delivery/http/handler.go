// Package http exposes the user platform's HTTP surface.
package http

import (
	"encoding/json"
	"net/http"
	"time"

	kernelauth "github.com/agile-factory/kernel/kernel/auth"
	"github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/kernel/permission"
	"github.com/agile-factory/kernel/kernel/response"
	"github.com/agile-factory/kernel/platform/user/domain"
	"github.com/agile-factory/kernel/platform/user/usecase"
)

// UserHandler handles HTTP requests for the user profile surface.
type UserHandler struct {
	getUC    *usecase.GetProfileUseCase
	updateUC *usecase.UpdateProfileUseCase
	checker  permission.Checker
}

// NewUserHandler wires the handler.
func NewUserHandler(
	getUC *usecase.GetProfileUseCase,
	updateUC *usecase.UpdateProfileUseCase,
	checker permission.Checker,
) *UserHandler {
	return &UserHandler{getUC: getUC, updateUC: updateUC, checker: checker}
}

// profileResponse is the JSON shape sent to clients.
type profileResponse struct {
	UserID      string `json:"userId"`
	DisplayName string `json:"displayName"`
	AvatarURL   string `json:"avatarUrl,omitempty"`
	Bio         string `json:"bio,omitempty"`
	Locale      string `json:"locale"`
	Timezone    string `json:"timezone"`
	UpdatedAt   string `json:"updatedAt"`
}

func toResponse(p *domain.UserProfile) profileResponse {
	return profileResponse{
		UserID:      p.UserID,
		DisplayName: p.DisplayName,
		AvatarURL:   p.AvatarURL,
		Bio:         p.Bio,
		Locale:      p.Locale,
		Timezone:    p.Timezone,
		UpdatedAt:   p.UpdatedAt.Format(time.RFC3339),
	}
}

// GetProfile handles GET /api/v1/users/{userId}/profile
//
// Any active tenant member with UserProfileView permission may read any profile.
// Profiles are created at signup via Subscribe(bus); GET returns 404 if absent.
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	callerID := kernelauth.UserIDFromContext(r.Context())
	if callerID == "" {
		response.Error(w, errors.Unauthorized("authentication required"))
		return
	}

	if err := h.checker.Require(r.Context(), permission.UserProfileView); err != nil {
		response.Error(w, err)
		return
	}

	targetUserID := r.PathValue("userId")
	if targetUserID == "" {
		response.Error(w, errors.BadRequest("userId path parameter is required"))
		return
	}

	out, err := h.getUC.Execute(r.Context(), usecase.GetProfileInput{
		UserID: targetUserID,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toResponse(out.Profile))
}

// patchProfileRequest is the JSON body for PATCH.
type patchProfileRequest struct {
	DisplayName *string `json:"displayName"`
	AvatarURL   *string `json:"avatarUrl"`
	Bio         *string `json:"bio"`
	Locale      *string `json:"locale"`
	Timezone    *string `json:"timezone"`
}

// UpdateProfile handles PATCH /api/v1/users/{userId}/profile
//
// A user may only update their own profile (callerID == pathUserId).
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	callerID := kernelauth.UserIDFromContext(r.Context())
	if callerID == "" {
		response.Error(w, errors.Unauthorized("authentication required"))
		return
	}

	targetUserID := r.PathValue("userId")
	if targetUserID == "" {
		response.Error(w, errors.BadRequest("userId path parameter is required"))
		return
	}

	// Users can only update their own profile.
	if callerID != targetUserID {
		response.Error(w, errors.Forbidden("you may only update your own profile"))
		return
	}

	if err := h.checker.Require(r.Context(), permission.UserProfileUpdateOwn); err != nil {
		response.Error(w, err)
		return
	}

	var body patchProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, errors.BadRequest("invalid JSON body"))
		return
	}

	out, err := h.updateUC.Execute(r.Context(), usecase.UpdateProfileInput{
		UserID:      targetUserID,
		DisplayName: body.DisplayName,
		AvatarURL:   body.AvatarURL,
		Bio:         body.Bio,
		Locale:      body.Locale,
		Timezone:    body.Timezone,
	})
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, toResponse(out.Profile))
}
