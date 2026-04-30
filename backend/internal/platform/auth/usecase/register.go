package usecase

import (
	"context"

	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/auth/domain"
)

// PasswordHasher defines how passwords should be hashed.
type PasswordHasher interface {
	Hash(password string) (string, error)
}

// RegisterUseCase handles the user registration flow.
type RegisterUseCase struct {
	repo   domain.UserRepository
	hasher PasswordHasher
}

// NewRegisterUseCase creates a new RegisterUseCase.
func NewRegisterUseCase(repo domain.UserRepository, hasher PasswordHasher) *RegisterUseCase {
	return &RegisterUseCase{
		repo:   repo,
		hasher: hasher,
	}
}

// Execute performs the registration.
func (uc *RegisterUseCase) Execute(ctx context.Context, id, tenantId, emailStr, password string) (*domain.User, error) {
	// Validate email
	email, err := domain.NewEmail(emailStr)
	if err != nil {
		return nil, err
	}

	// Check if user already exists
	existingUser, _ := uc.repo.FindByEmail(ctx, email)
	if existingUser != nil {
		return nil, domain.ErrUserAlreadyExists
	}

	// Hash password
	hashedPassword, err := uc.hasher.Hash(password)
	if err != nil {
		return nil, err
	}

	// Create new user (defaulting to viewer role, or agent depending on business logic)
	user := domain.NewUser(id, tenantId, email, hashedPassword, domain.ROLE_VIEWER)

	// Save to DB
	if err := uc.repo.Save(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}
