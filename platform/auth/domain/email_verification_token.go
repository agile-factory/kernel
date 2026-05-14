package domain

import (
	"context"
	"time"
)

// EmailVerificationToken proves email address ownership at signup. The raw
// token IS the secret; we store it directly (single-use, 24h lifetime).
type EmailVerificationToken struct {
	ID         string
	UserID     string
	Token      string
	ExpiresAt  time.Time
	ConsumedAt *time.Time
	CreatedAt  time.Time
}

// NewEmailVerificationToken builds a token aggregate. The caller is
// responsible for generating the opaque random Token value.
func NewEmailVerificationToken(id, userID, token string, ttl time.Duration) *EmailVerificationToken {
	now := time.Now().UTC()
	return &EmailVerificationToken{
		ID:        id,
		UserID:    userID,
		Token:     token,
		ExpiresAt: now.Add(ttl),
		CreatedAt: now,
	}
}

// IsRedeemable returns true when the token can be consumed.
func (t *EmailVerificationToken) IsRedeemable(now time.Time) bool {
	return t.ConsumedAt == nil && now.Before(t.ExpiresAt)
}

// EmailVerificationTokenRepository is the persistence contract.
type EmailVerificationTokenRepository interface {
	Save(ctx context.Context, t *EmailVerificationToken) error
	FindByToken(ctx context.Context, token string) (*EmailVerificationToken, error)
	Consume(ctx context.Context, id string) error
}
