package usecase

import (
	"context"

	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/kernel/errors"
	auditdomain "github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/audit/domain"
	auditusecase "github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/audit/usecase"
	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/auth/domain"
)

// PasswordVerifier defines how passwords should be verified.
type PasswordVerifier interface {
	Verify(hashedPassword, password string) error
}

// TokenGenerator defines how JWT tokens are created.
type TokenGenerator interface {
	GenerateToken(user *domain.User) (string, error)
}

// LoginUseCase coordinates user authentication and token generation.
type LoginUseCase struct {
	repo       domain.UserRepository
	verifier   PasswordVerifier
	jwtService TokenGenerator
	audit      *auditusecase.LogEventUseCase
}

// NewLoginUseCase creates a new LoginUseCase.
func NewLoginUseCase(repo domain.UserRepository, verifier PasswordVerifier, jwt TokenGenerator, audit *auditusecase.LogEventUseCase) *LoginUseCase {
	return &LoginUseCase{
		repo:       repo,
		verifier:   verifier,
		jwtService: jwt,
		audit:      audit,
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
	token, err := uc.jwtService.GenerateToken(user)
	if err != nil {
		return "", nil, errors.WrapInternal(err)
	}

	uc.audit.Execute(ctx, auditusecase.LogEventParams{
		TenantID:   "", // Login is not tenant-scoped right now
		ActorID:    user.ID,
		Action:     auditdomain.ACTION_USER_LOGGED_IN,
		TargetType: "user",
		TargetID:   user.ID,
		Metadata: map[string]any{
			"email": user.Email,
		},
	})

	return token, user, nil
}
