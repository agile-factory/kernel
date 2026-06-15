package http

import (
	"errors"
	"strings"
	"time"

	"github.com/agile-factory/kernel/platform/membership/domain"
)

// ─── Invite ──────────────────────────────────────────────────────────────────

// InviteRequest is the payload for inviting a member by email — RFC-0018 §6.
// The invitee may not yet have a user account; AcceptInvite will create one.
type InviteRequest struct {
	TenantID string `json:"tenantId"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// Validate checks InviteRequest fields.
func (r *InviteRequest) Validate() error {
	if strings.TrimSpace(r.TenantID) == "" {
		return errors.New("tenantId is required")
	}
	if strings.TrimSpace(r.Email) == "" {
		return errors.New("email is required")
	}
	switch domain.MemberRole(r.Role) {
	case domain.MEMBER_ROLE_ADMIN, domain.MEMBER_ROLE_AGENT, domain.MEMBER_ROLE_VIEWER:
		// valid — ROLE_OWNER is intentionally absent (RFC-0018 §6)
	default:
		return errors.New("role must be ROLE_ADMIN, ROLE_AGENT, or ROLE_VIEWER")
	}
	return nil
}

// InviteResponse echoes the persisted invite plus the raw token. The raw
// token is intended to be emailed by the notification platform (consumer of
// `member.invited`); the API returns it so callers can implement custom
// flows (in-house mailer, copy-to-clipboard for dev, …) without round-trips.
type InviteResponse struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenantId"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Token     string `json:"token"`
	ExpiresAt string `json:"expiresAt"`
	CreatedAt string `json:"createdAt"`
}

func toInviteResponse(invite *domain.InviteToken, rawToken string) InviteResponse {
	return InviteResponse{
		ID:        invite.ID,
		TenantID:  invite.TenantID,
		Email:     invite.Email,
		Role:      string(invite.Role),
		Token:     rawToken,
		ExpiresAt: invite.ExpiresAt.Format(time.RFC3339),
		CreatedAt: invite.CreatedAt.Format(time.RFC3339),
	}
}

// ─── Membership listing / revocation ─────────────────────────────────────────

// MembershipResponse is the public representation of a membership.
type MembershipResponse struct {
	ID        string  `json:"id"`
	UserID    string  `json:"userId"`
	TenantID  string  `json:"tenantId"`
	Role      string  `json:"role"`
	Status    string  `json:"status"`
	InvitedBy string  `json:"invitedBy,omitempty"`
	JoinedAt  *string `json:"joinedAt,omitempty"`
	CreatedAt string  `json:"createdAt"`
}

// toMembershipResponse converts a domain Membership to the HTTP response DTO.
func toMembershipResponse(m *domain.Membership) MembershipResponse {
	resp := MembershipResponse{
		ID:        m.ID,
		UserID:    m.UserID,
		TenantID:  m.TenantID,
		Role:      string(m.Role),
		Status:    string(m.Status),
		InvitedBy: m.InvitedBy,
		CreatedAt: m.CreatedAt.Format(time.RFC3339),
	}
	if m.JoinedAt != nil {
		s := m.JoinedAt.Format(time.RFC3339)
		resp.JoinedAt = &s
	}
	return resp
}
