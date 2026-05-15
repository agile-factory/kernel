package domain

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"time"
)

// RefreshToken is an opaque long-lived credential the client trades for a
// fresh access token. Only the hex SHA-256 of the raw token is stored —
// the raw value lives only on the wire and in the client. Per RFC-0018 §5.
//
// TenantID is optional: SignupOwner / AcceptInvite produce a tenant-scoped
// token (the just-created / just-joined tenant), but a plain Login response
// is not tenant-scoped — the caller picks a tenant per-request via the JWT.
type RefreshToken struct {
	ID        string
	UserID    string
	TenantID  string // "" when the token is not tenant-scoped
	TokenHash string // hex SHA-256 of the opaque token
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

// IsActive returns true when the token can still be redeemed.
func (t *RefreshToken) IsActive(now time.Time) bool {
	return t.RevokedAt == nil && now.Before(t.ExpiresAt)
}

// NewRefreshToken mints a fresh opaque token + its stored hash. Returns the
// raw token (to send to the client) and the persisted aggregate.
//
// The raw token is 32 bytes of crypto/rand encoded base64-url — 256 bits of
// entropy, URL-safe, no padding.
func NewRefreshToken(id, userID, tenantID string, ttl time.Duration) (raw string, rt *RefreshToken, err error) {
	raw, err = generateOpaqueToken()
	if err != nil {
		return "", nil, err
	}
	hash := HashRefreshToken(raw)
	now := time.Now().UTC()
	return raw, &RefreshToken{
		ID:        id,
		UserID:    userID,
		TenantID:  tenantID,
		TokenHash: hash,
		ExpiresAt: now.Add(ttl),
		CreatedAt: now,
	}, nil
}

// HashRefreshToken returns the hex SHA-256 of raw — the value stored in
// refresh_tokens.token_hash and matched at refresh time.
func HashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func generateOpaqueToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

// RefreshTokenRepository is the persistence contract for refresh tokens.
type RefreshTokenRepository interface {
	Save(ctx context.Context, t *RefreshToken) error
	FindByHash(ctx context.Context, hash string) (*RefreshToken, error)
	Revoke(ctx context.Context, id string) error
}
