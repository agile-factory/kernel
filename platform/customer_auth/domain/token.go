package domain

import "time"

// CustomerRefreshToken is a long-lived opaque token for session renewal.
// Follows the rolling-refresh pattern (RFC-0018 §5).
type CustomerRefreshToken struct {
	ID            string
	EndCustomerID string
	TenantID      string
	TokenHash     string     // SHA-256 of the opaque bearer value
	ExpiresAt     time.Time
	RevokedAt     *time.Time
	CreatedAt     time.Time
}

// IsValid returns true when the token is neither revoked nor expired.
func (t *CustomerRefreshToken) IsValid(now time.Time) bool {
	return t.RevokedAt == nil && t.ExpiresAt.After(now)
}

// CustomerResetToken is a single-use password-reset token (OTP or email link).
type CustomerResetToken struct {
	ID            string
	EndCustomerID string
	TokenHash     string     // SHA-256 of the opaque value sent to the user
	ExpiresAt     time.Time
	ConsumedAt    *time.Time
	CreatedAt     time.Time
}

// IsValid returns true when the token is neither consumed nor expired.
func (t *CustomerResetToken) IsValid(now time.Time) bool {
	return t.ConsumedAt == nil && t.ExpiresAt.After(now)
}
