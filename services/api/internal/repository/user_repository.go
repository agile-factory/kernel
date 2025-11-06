package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yourusername/immobilier-saas/services/api/internal/domain"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user with this email already exists")
	ErrTokenNotFound     = errors.New("token not found")
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (
			id, email, password_hash, first_name, last_name, phone,
			role, status, email_verified, phone_verified,
			verification_token, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)
	`

	user.ID = uuid.New()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	user.Status = domain.StatusPendingVerification
	user.EmailVerified = false
	user.PhoneVerified = false

	_, err := r.db.ExecContext(
		ctx,
		query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.FirstName,
		user.LastName,
		user.Phone,
		user.Role,
		user.Status,
		user.EmailVerified,
		user.PhoneVerified,
		user.VerificationToken,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		// Check for unique constraint violation
		if err.Error() == "pq: duplicate key value violates unique constraint \"users_email_key\"" {
			return ErrUserAlreadyExists
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT
			id, email, password_hash, first_name, last_name, phone,
			role, status, email_verified, phone_verified,
			verification_token, reset_password_token, reset_password_expires,
			last_login, created_at, updated_at, deleted_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`

	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.Phone,
		&user.Role,
		&user.Status,
		&user.EmailVerified,
		&user.PhoneVerified,
		&user.VerificationToken,
		&user.ResetPasswordToken,
		&user.ResetPasswordExpires,
		&user.LastLogin,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}

	return user, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT
			id, email, password_hash, first_name, last_name, phone,
			role, status, email_verified, phone_verified,
			verification_token, reset_password_token, reset_password_expires,
			last_login, created_at, updated_at, deleted_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`

	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.Phone,
		&user.Role,
		&user.Status,
		&user.EmailVerified,
		&user.PhoneVerified,
		&user.VerificationToken,
		&user.ResetPasswordToken,
		&user.ResetPasswordExpires,
		&user.LastLogin,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

// Update updates user information
func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users
		SET
			first_name = $2,
			last_name = $3,
			phone = $4,
			updated_at = $5
		WHERE id = $1 AND deleted_at IS NULL
	`

	user.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(
		ctx,
		query,
		user.ID,
		user.FirstName,
		user.LastName,
		user.Phone,
		user.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// UpdateStatus updates user status
func (r *UserRepository) UpdateStatus(ctx context.Context, userID uuid.UUID, status domain.UserStatus) error {
	query := `
		UPDATE users
		SET status = $2, updated_at = $3
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, userID, status, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update user status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// UpdateEmailVerified marks email as verified
func (r *UserRepository) UpdateEmailVerified(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE users
		SET
			email_verified = true,
			status = $2,
			verification_token = NULL,
			updated_at = $3
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		userID,
		domain.StatusActive,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to update email verified: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// SetResetPasswordToken sets the password reset token and expiry
func (r *UserRepository) SetResetPasswordToken(ctx context.Context, email, token string, expiresAt time.Time) error {
	query := `
		UPDATE users
		SET
			reset_password_token = $2,
			reset_password_expires = $3,
			updated_at = $4
		WHERE email = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, email, token, expiresAt, time.Now())
	if err != nil {
		return fmt.Errorf("failed to set reset password token: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// GetByResetToken retrieves user by reset password token
func (r *UserRepository) GetByResetToken(ctx context.Context, token string) (*domain.User, error) {
	query := `
		SELECT
			id, email, password_hash, first_name, last_name, phone,
			role, status, email_verified, phone_verified,
			verification_token, reset_password_token, reset_password_expires,
			last_login, created_at, updated_at, deleted_at
		FROM users
		WHERE reset_password_token = $1
			AND reset_password_expires > $2
			AND deleted_at IS NULL
	`

	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, token, time.Now()).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.Phone,
		&user.Role,
		&user.Status,
		&user.EmailVerified,
		&user.PhoneVerified,
		&user.VerificationToken,
		&user.ResetPasswordToken,
		&user.ResetPasswordExpires,
		&user.LastLogin,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrTokenNotFound
		}
		return nil, fmt.Errorf("failed to get user by reset token: %w", err)
	}

	return user, nil
}

// UpdatePassword updates user password and clears reset token
func (r *UserRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	query := `
		UPDATE users
		SET
			password_hash = $2,
			reset_password_token = NULL,
			reset_password_expires = NULL,
			updated_at = $3
		WHERE id = $1 AND deleted_at IS NULL
	`

	result, err := r.db.ExecContext(ctx, query, userID, passwordHash, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// UpdateLastLogin updates the last login timestamp
func (r *UserRepository) UpdateLastLogin(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE users
		SET last_login = $2, updated_at = $3
		WHERE id = $1 AND deleted_at IS NULL
	`

	now := time.Now()
	result, err := r.db.ExecContext(ctx, query, userID, now, now)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// GetByVerificationToken retrieves user by verification token
func (r *UserRepository) GetByVerificationToken(ctx context.Context, token string) (*domain.User, error) {
	query := `
		SELECT
			id, email, password_hash, first_name, last_name, phone,
			role, status, email_verified, phone_verified,
			verification_token, reset_password_token, reset_password_expires,
			last_login, created_at, updated_at, deleted_at
		FROM users
		WHERE verification_token = $1 AND deleted_at IS NULL
	`

	user := &domain.User{}
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.Phone,
		&user.Role,
		&user.Status,
		&user.EmailVerified,
		&user.PhoneVerified,
		&user.VerificationToken,
		&user.ResetPasswordToken,
		&user.ResetPasswordExpires,
		&user.LastLogin,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrTokenNotFound
		}
		return nil, fmt.Errorf("failed to get user by verification token: %w", err)
	}

	return user, nil
}

// Refresh Token methods

// CreateRefreshToken stores a new refresh token
func (r *UserRepository) CreateRefreshToken(ctx context.Context, token *domain.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (
			id, user_id, token_hash, device_info, ip_address, expires_at, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)
	`

	token.ID = uuid.New()
	token.CreatedAt = time.Now()

	_, err := r.db.ExecContext(
		ctx,
		query,
		token.ID,
		token.UserID,
		token.TokenHash,
		token.DeviceInfo,
		token.IPAddress,
		token.ExpiresAt,
		token.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create refresh token: %w", err)
	}

	return nil
}

// GetRefreshToken retrieves a refresh token by hash
func (r *UserRepository) GetRefreshToken(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, device_info, ip_address, expires_at, created_at
		FROM refresh_tokens
		WHERE token_hash = $1 AND expires_at > $2
	`

	token := &domain.RefreshToken{}
	err := r.db.QueryRowContext(ctx, query, tokenHash, time.Now()).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.DeviceInfo,
		&token.IPAddress,
		&token.ExpiresAt,
		&token.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrTokenNotFound
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	return token, nil
}

// RevokeRefreshToken deletes a refresh token
func (r *UserRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	query := `DELETE FROM refresh_tokens WHERE token_hash = $1`

	_, err := r.db.ExecContext(ctx, query, tokenHash)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	return nil
}

// RevokeAllUserTokens deletes all refresh tokens for a user
func (r *UserRepository) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM refresh_tokens WHERE user_id = $1`

	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke all user tokens: %w", err)
	}

	return nil
}

// CleanupExpiredTokens removes expired refresh tokens
func (r *UserRepository) CleanupExpiredTokens(ctx context.Context) error {
	query := `DELETE FROM refresh_tokens WHERE expires_at < $1`

	_, err := r.db.ExecContext(ctx, query, time.Now())
	if err != nil {
		return fmt.Errorf("failed to cleanup expired tokens: %w", err)
	}

	return nil
}
