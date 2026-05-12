package usecase

import (
	"context"

	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/auth/domain"
)

// TokenValidator defines how tokens are validated.
type TokenValidator interface {
	ValidateToken(token string) (string, error) // Returns UserID
}

// RefreshUseCase handles the token refresh flow.
type RefreshUseCase struct {
	repo           domain.UserRepository
	validator      TokenValidator
	tokenGenerator TokenGenerator
}

// NewRefreshUseCase creates a new RefreshUseCase.
func NewRefreshUseCase(repo domain.UserRepository, validator TokenValidator, tokenGenerator TokenGenerator) *RefreshUseCase {
	return &RefreshUseCase{
		repo:           repo,
		validator:      validator,
		tokenGenerator: tokenGenerator,
	}
}

// Execute validates an old token and returns a new one if the user still exists.
func (uc *RefreshUseCase) Execute(ctx context.Context, token string) (string, error) {
	// Validate old token
	userID, err := uc.validator.ValidateToken(token)
	if err != nil {
		return "", err
	}

	user, err := uc.repo.FindByID(ctx, userID)
	if err != nil {
		return "", domain.ErrUserNotFound
	}

	return uc.tokenGenerator.GenerateToken(user)
}
