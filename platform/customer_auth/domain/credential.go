// Package domain holds the customer_auth platform's domain types and contracts.
package domain

import "time"

// CustomerCredential holds the authentication secrets for an end-customer.
// One row per end-customer; linked via end_customer_id.
type CustomerCredential struct {
	ID             string
	EndCustomerID  string
	TenantID       string
	PasswordHash   string     // bcrypt hash (max 72 bytes input for bcrypt safety)
	LastLoginAt    *time.Time
	FailedAttempts int
	LockedUntil    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// IsLocked returns true when the account is currently locked.
// now is injected for deterministic testing — never call time.Now() in domain.
func (c *CustomerCredential) IsLocked(now time.Time) bool {
	return c.LockedUntil != nil && c.LockedUntil.After(now)
}

// RecordLoginFailure increments failed_attempts and locks the account when the
// threshold is reached (5 attempts → 15 minutes lockout).
func (c *CustomerCredential) RecordLoginFailure(now time.Time) {
	c.FailedAttempts++
	if c.FailedAttempts >= 5 {
		lockUntil := now.Add(15 * time.Minute)
		c.LockedUntil = &lockUntil
	}
	c.UpdatedAt = now
}

// RecordLoginSuccess resets the failure counters and updates last_login_at.
func (c *CustomerCredential) RecordLoginSuccess(now time.Time) {
	c.FailedAttempts = 0
	c.LockedUntil = nil
	c.LastLoginAt = &now
	c.UpdatedAt = now
}
