// Package usecase contains the user platform's application use cases.
package usecase

import (
	"context"

	apperrors "github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/platform/user/domain"
)

// GetProfileInput carries what the caller needs to supply.
type GetProfileInput struct {
	UserID string
	// Email is used to initialise a DisplayName when no profile row exists yet.
	// Callers should pass the authenticated user's email; leave empty only if the
	// caller has no access to it (GET by another member will not lazy-init).
	Email string
}

// GetProfileOutput wraps the result.
type GetProfileOutput struct {
	Profile *domain.UserProfile
}

// GetProfileUseCase retrieves a user profile, creating it lazily when absent.
type GetProfileUseCase struct {
	repo domain.UserProfileRepository
}

// NewGetProfileUseCase builds the use case.
func NewGetProfileUseCase(repo domain.UserProfileRepository) *GetProfileUseCase {
	return &GetProfileUseCase{repo: repo}
}

// Execute fetches the profile for input.UserID.
// If no row exists and input.Email is non-empty, a new profile is persisted and
// returned. If no row exists and email is empty, errors.NotFound is returned.
func (uc *GetProfileUseCase) Execute(ctx context.Context, input GetProfileInput) (GetProfileOutput, error) {
	if input.UserID == "" {
		return GetProfileOutput{}, apperrors.BadRequest("userID is required")
	}

	profile, err := uc.repo.FindByUserID(ctx, input.UserID)
	if err != nil {
		appErr, ok := err.(*apperrors.AppError)
		if !ok || appErr.HTTPCode != 404 {
			return GetProfileOutput{}, err
		}
		// Profile absent — lazy-init if email provided.
		if input.Email == "" {
			return GetProfileOutput{}, err // propagate 404
		}
		profile = domain.NewFromEmail(input.UserID, input.Email)
		if upsertErr := uc.repo.Upsert(ctx, profile); upsertErr != nil {
			return GetProfileOutput{}, upsertErr
		}
	}

	return GetProfileOutput{Profile: profile}, nil
}
