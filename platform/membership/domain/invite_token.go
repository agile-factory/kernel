package domain

import (
	"context"
	"time"
)

// InviteToken is issued when an admin invites a new member to a tenant. The
// invitee redeems it through POST /api/v1/auth/accept-invite, which creates
// the user (if new) and an ACTIVE membership with the pre-decided role. Per
// RFC-0018 §6.
//
// The raw token IS the secret (single-use, 7-day lifetime). It is opaque to
// the consumer and never reused: redeeming twice surfaces as
// ErrInviteTokenAlreadyConsumed.
type InviteToken struct {
	ID         string
	TenantID   string
	InvitedBy  string
	Email      string
	Role       MemberRole
	Token      string
	ExpiresAt  time.Time
	ConsumedAt *time.Time
	RevokedAt  *time.Time
	CreatedAt  time.Time
}

// NewInviteToken builds the aggregate. ROLE_OWNER is rejected — ownership
// is established at tenant creation and never assigned via invitation.
func NewInviteToken(id, tenantID, invitedBy, email string, role MemberRole, token string, ttl time.Duration) (*InviteToken, error) {
	if role == MEMBER_ROLE_OWNER {
		return nil, ErrCannotInviteAsOwner
	}
	now := time.Now().UTC()
	return &InviteToken{
		ID:        id,
		TenantID:  tenantID,
		InvitedBy: invitedBy,
		Email:     email,
		Role:      role,
		Token:     token,
		ExpiresAt: now.Add(ttl),
		CreatedAt: now,
	}, nil
}

// IsRedeemable returns true when the token can still be consumed.
func (t *InviteToken) IsRedeemable(now time.Time) bool {
	return t.ConsumedAt == nil && t.RevokedAt == nil && now.Before(t.ExpiresAt)
}

// InviteTokenRepository is the persistence contract for invite tokens.
type InviteTokenRepository interface {
	Save(ctx context.Context, t *InviteToken) error
	FindByToken(ctx context.Context, token string) (*InviteToken, error)
	FindPendingByTenantAndEmail(ctx context.Context, tenantID, email string) (*InviteToken, error)
	Consume(ctx context.Context, id string) error
	Revoke(ctx context.Context, id string) error
}
