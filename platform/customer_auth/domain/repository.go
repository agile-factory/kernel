package domain

import (
	"context"
	"time"
)

// CredentialRepository is the persistence contract for CustomerCredential.
type CredentialRepository interface {
	// Create inserts a new credential row. Returns ErrCredentialAlreadyExists on conflict.
	Create(ctx context.Context, cred *CustomerCredential) error

	// FindByEndCustomerID returns the credential for the given end-customer.
	// Returns ErrCredentialNotFound when absent.
	FindByEndCustomerID(ctx context.Context, endCustomerID string) (*CustomerCredential, error)

	// FindByEmail looks up a credential via the end_customers.email column (JOIN).
	// Returns ErrCredentialNotFound when absent — callers surface ErrInvalidCredentials
	// to prevent email enumeration.
	FindByEmail(ctx context.Context, email, tenantID string) (*CustomerCredential, error)

	// Update persists all mutable fields (password_hash, failed_attempts, locked_until, etc.).
	Update(ctx context.Context, cred *CustomerCredential) error
}

// CustomerRefreshTokenRepository manages session refresh tokens.
type CustomerRefreshTokenRepository interface {
	// Create inserts a new refresh token row.
	Create(ctx context.Context, t *CustomerRefreshToken) error

	// FindByHash looks up an active (non-revoked) refresh token by its SHA-256 hash.
	// Returns ErrTokenNotFound when absent or revoked.
	FindByHash(ctx context.Context, tokenHash string) (*CustomerRefreshToken, error)

	// Revoke marks a specific refresh token as revoked.
	Revoke(ctx context.Context, id string) error

	// RevokeAllForEndCustomer revokes all refresh tokens for the given end-customer.
	RevokeAllForEndCustomer(ctx context.Context, endCustomerID string) error
}

// CustomerResetTokenRepository manages password-reset tokens.
type CustomerResetTokenRepository interface {
	// Create inserts a new reset token row.
	Create(ctx context.Context, t *CustomerResetToken) error

	// FindByHash looks up a valid (non-consumed, non-expired) reset token.
	// Returns ErrTokenNotFound when absent.
	FindByHash(ctx context.Context, tokenHash string) (*CustomerResetToken, error)

	// MarkConsumed sets consumed_at = now on the token.
	MarkConsumed(ctx context.Context, id string, now time.Time) error

	// HasRecentToken returns true when an unconsumed reset token for the given
	// end-customer was created within the last sinceSeconds seconds.
	// Used to enforce the OTP 60-second rate-limit.
	HasRecentToken(ctx context.Context, endCustomerID string, since time.Time) (bool, error)
}
