package usecase

import (
	"context"
	"time"

	"github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/platform/audit"
	auditdomain "github.com/agile-factory/kernel/platform/audit/domain"
	"github.com/agile-factory/kernel/platform/auth/domain"
	"github.com/google/uuid"
)

// PasswordHasher hashes a plaintext password for storage.
type PasswordHasher interface {
	Hash(password string) (string, error)
}

// PasswordVerifier compares a plaintext password against a stored hash.
type PasswordVerifier interface {
	Verify(hashedPassword, password string) error
}

// TokenGenerator emits a JWT access token bound to a user.
type TokenGenerator interface {
	GenerateToken(user *domain.User) (string, error)
}

// dummyPasswordHash is a bcrypt hash (cost 10, the default) of a random
// string nobody knows. Login verifies against it when the email is unknown.
const dummyPasswordHash = "$2a$10$xGrkO4RvYbbgUD2kLDM34O.mG4QzZU1fCF76UQixoey.J4X2UT7RO"

// LoginOutput bundles the access + refresh tokens issued at login. The
// refresh token is NOT tenant-scoped: the caller picks a tenant per-request
// via the JWT (or a dedicated header / path) once they are signed in.
type LoginOutput struct {
	AccessToken  string
	RefreshToken string
	User         *domain.User
}

// LoginUseCase coordinates user authentication and token generation.
type LoginUseCase struct {
	users           domain.UserRepository
	refreshTokens   domain.RefreshTokenRepository
	verifier        PasswordVerifier
	jwtService      TokenGenerator
	auditLogger     audit.EventLogger
	refreshTokenTTL time.Duration
}

// NewLoginUseCase creates a new LoginUseCase. refreshTokens may be nil during
// transition; when nil, LoginOutput.RefreshToken is left empty.
func NewLoginUseCase(
	users domain.UserRepository,
	refreshTokens domain.RefreshTokenRepository,
	verifier PasswordVerifier,
	jwt TokenGenerator,
	auditLogger audit.EventLogger,
	refreshTokenTTL time.Duration,
) *LoginUseCase {
	return &LoginUseCase{
		users:           users,
		refreshTokens:   refreshTokens,
		verifier:        verifier,
		jwtService:      jwt,
		auditLogger:     auditLogger,
		refreshTokenTTL: refreshTokenTTL,
	}
}

// Execute authenticates the user and emits an access + refresh token pair.
//
// Returns ErrInvalidCredentials on any auth failure (unknown email / wrong
// password / invalid format) to avoid user enumeration. Returns
// ErrUserNotActive when the credentials match but the account is
// PENDING_EMAIL / SUSPENDED / DELETED.
func (uc *LoginUseCase) Execute(ctx context.Context, emailStr, password string) (*LoginOutput, error) {
	email, err := domain.NewEmail(emailStr)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	user, err := uc.users.FindByEmail(ctx, email)
	if err != nil {
		// Burn a comparable amount of CPU so an unknown email is not
		// distinguishable from a wrong password by response time.
		_ = uc.verifier.Verify(dummyPasswordHash, password)
		return nil, domain.ErrInvalidCredentials
	}
	if err := uc.verifier.Verify(user.HashedPassword, password); err != nil {
		return nil, domain.ErrInvalidCredentials
	}
	if !user.IsActive() {
		return nil, domain.ErrUserNotActive
	}

	accessToken, err := uc.jwtService.GenerateToken(user)
	if err != nil {
		return nil, errors.WrapInternal(err)
	}

	out := &LoginOutput{User: user, AccessToken: accessToken}

	if uc.refreshTokens != nil {
		rawRefresh, refresh, err := domain.NewRefreshToken(uuid.NewString(), user.ID, "", uc.refreshTokenTTL)
		if err != nil {
			return nil, errors.WrapInternal(err)
		}
		if err := uc.refreshTokens.Save(ctx, refresh); err != nil {
			return nil, errors.WrapInternal(err)
		}
		out.RefreshToken = rawRefresh
	}

	if uc.auditLogger != nil {
		uc.auditLogger.Execute(ctx, audit.LogEventParams{
			TenantID:   "", // Login is not tenant-scoped
			ActorID:    user.ID,
			Action:     auditdomain.ACTION_USER_LOGGED_IN,
			TargetType: "user",
			TargetID:   user.ID,
			Metadata: map[string]any{
				"email": user.Email.String(),
			},
		})
	}

	return out, nil
}
