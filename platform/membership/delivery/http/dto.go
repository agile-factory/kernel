package http

import (
	"errors"
	"strings"
	"time"

	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/membership/domain"
)

// ─── Invite ──────────────────────────────────────────────────────────────────

// InviteRequest is the payload for inviting a user to a tenant.
type InviteRequest struct {
	TenantID      string `json:"tenantId"`
	InviteeUserID string `json:"inviteeUserId"`
	Role          string `json:"role"`
}

// Validate checks InviteRequest fields.
func (r *InviteRequest) Validate() error {
	if strings.TrimSpace(r.TenantID) == "" {
		return errors.New("tenantId is required")
	}
	if strings.TrimSpace(r.InviteeUserID) == "" {
		return errors.New("inviteeUserId is required")
	}
	switch domain.MemberRole(r.Role) {
	case domain.MEMBER_ROLE_ADMIN, domain.MEMBER_ROLE_AGENT, domain.MEMBER_ROLE_VIEWER:
		// valid
	default:
		return errors.New("role must be ROLE_ADMIN, ROLE_AGENT, or ROLE_VIEWER")
	}
	return nil
}

// ─── Response ─────────────────────────────────────────────────────────────────

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
