package usecase

import (
	"context"

	"github.com/AgileFactory/kernel/kernel/errors"
	"github.com/AgileFactory/kernel/platform/audit"
	auditdomain "github.com/AgileFactory/kernel/platform/audit/domain"
	"github.com/AgileFactory/kernel/platform/auth/domain"
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
	repo        domain.UserRepository
	verifier    PasswordVerifier
	jwtService  TokenGenerator
	auditLogger audit.EventLogger
}

// NewLoginUseCase creates a new LoginUseCase.
func NewLoginUseCase(repo domain.UserRepository, verifier PasswordVerifier, jwt TokenGenerator, auditLogger audit.EventLogger) *LoginUseCase {
	return &LoginUseCase{
		repo:        repo,
		verifier:    verifier,
		jwtService:  jwt,
		auditLogger: auditLogger,
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

	uc.auditLogger.Execute(ctx, audit.LogEventParams{
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
