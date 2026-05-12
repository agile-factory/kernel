package usecase

import (
	"context"

	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/auth/domain"
)

// PasswordVerifier defines how passwords should be verified against a hash.
type PasswordVerifier interface {
	Verify(hashedPassword, password string) error
}

// TokenGenerator defines how tokens (JWT) are generated.
type TokenGenerator interface {
	GenerateToken(user *domain.User) (string, error)
}

// LoginUseCase handles the user login flow.
type LoginUseCase struct {
	repo           domain.UserRepository
	verifier       PasswordVerifier
	tokenGenerator TokenGenerator
}

// NewLoginUseCase creates a new LoginUseCase.
func NewLoginUseCase(repo domain.UserRepository, verifier PasswordVerifier, tokenGenerator TokenGenerator) *LoginUseCase {
	return &LoginUseCase{
		repo:           repo,
		verifier:       verifier,
		tokenGenerator: tokenGenerator,
	}
}

// Execute performs the login and returns a token.
func (uc *LoginUseCase) Execute(ctx context.Context, emailStr, password string) (string, *domain.User, error) {
	// Validate email format
	email, err := domain.NewEmail(emailStr)
	if err != nil {
		return "", nil, domain.ErrInvalidCredentials
	}

	// Find user
	user, err := uc.repo.FindByEmail(ctx, email)
	if err != nil {
		// Even if not found, we return InvalidCredentials to avoid user enumeration
		return "", nil, domain.ErrInvalidCredentials
	}

	// Verify password
	if err := uc.verifier.Verify(user.HashedPassword, password); err != nil {
		return "", nil, domain.ErrInvalidCredentials
	}

	// Generate token
	token, err := uc.tokenGenerator.GenerateToken(user)
	if err != nil {
		return "", nil, err
	}

	return token, user, nil
}
