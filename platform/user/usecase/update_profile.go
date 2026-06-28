package usecase

import (
	"context"
	"time"

	apperrors "github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/platform/user/domain"
)

// UpdateProfileInput carries the partial update. Nil pointer fields are not applied.
type UpdateProfileInput struct {
	UserID      string
	Email       string  // for lazy-init if profile absent
	DisplayName *string
	AvatarURL   *string
	Bio         *string
	Locale      *string
	Timezone    *string
}

// UpdateProfileOutput wraps the updated profile.
type UpdateProfileOutput struct {
	Profile *domain.UserProfile
}

// UpdateProfileUseCase applies a partial update to a user profile.
type UpdateProfileUseCase struct {
	repo domain.UserProfileRepository
}

// NewUpdateProfileUseCase builds the use case.
func NewUpdateProfileUseCase(repo domain.UserProfileRepository) *UpdateProfileUseCase {
	return &UpdateProfileUseCase{repo: repo}
}

// Execute fetches the profile (lazy-inits if absent), applies non-nil fields,
// persists, and returns the updated profile.
func (uc *UpdateProfileUseCase) Execute(ctx context.Context, input UpdateProfileInput) (UpdateProfileOutput, error) {
	if input.UserID == "" {
		return UpdateProfileOutput{}, apperrors.BadRequest("userID is required")
	}

	profile, err := uc.repo.FindByUserID(ctx, input.UserID)
	if err != nil {
		appErr, ok := err.(*apperrors.AppError)
		if !ok || appErr.HTTPCode != 404 {
			return UpdateProfileOutput{}, err
		}
		// Lazy-init
		email := input.Email
		if email == "" {
			email = input.UserID // fallback: use ID as placeholder name
		}
		profile = domain.NewFromEmail(input.UserID, email)
	}

	// Apply non-nil fields
	if input.DisplayName != nil {
		profile.DisplayName = *input.DisplayName
	}
	if input.AvatarURL != nil {
		profile.AvatarURL = *input.AvatarURL
	}
	if input.Bio != nil {
		profile.Bio = *input.Bio
	}
	if input.Locale != nil {
		profile.Locale = *input.Locale
	}
	if input.Timezone != nil {
		profile.Timezone = *input.Timezone
	}
	profile.UpdatedAt = time.Now().UTC()

	if err := uc.repo.Upsert(ctx, profile); err != nil {
		return UpdateProfileOutput{}, err
	}

	return UpdateProfileOutput{Profile: profile}, nil
}
