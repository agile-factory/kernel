package domain

import "time"

// MemberRole defines the authorization level within a tenant.
type MemberRole string

const (
	// MEMBER_ROLE_OWNER is assigned to the founding member of a tenant (the creator).
	// An owner cannot be revoked by other admins and holds all administrative privileges.
	MEMBER_ROLE_OWNER  MemberRole = "ROLE_OWNER"
	MEMBER_ROLE_ADMIN  MemberRole = "ROLE_ADMIN"
	MEMBER_ROLE_AGENT  MemberRole = "ROLE_AGENT"
	MEMBER_ROLE_VIEWER MemberRole = "ROLE_VIEWER"
)

// MemberStatus represents the lifecycle state of a membership.
type MemberStatus string

const (
	MEMBER_STATUS_PENDING MemberStatus = "PENDING"
	MEMBER_STATUS_ACTIVE  MemberStatus = "ACTIVE"
	MEMBER_STATUS_REVOKED MemberStatus = "REVOKED"
)

// Membership is the join aggregate between a User and a Tenant.
// It carries its own role and lifecycle independently of the user entity.
type Membership struct {
	ID         string
	UserID     string
	TenantID   string
	Role       MemberRole
	Status     MemberStatus
	InvitedBy  string     // empty string when self-registered
	JoinedAt   *time.Time // nil until accepted
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// NewSelfMembership creates an ACTIVE membership for a self-registered user.
// There is no inviter, and joined_at is set immediately to now.
func NewSelfMembership(id, userID, tenantID string, role MemberRole) *Membership {
	now := time.Now()
	return &Membership{
		ID:        id,
		UserID:    userID,
		TenantID:  tenantID,
		Role:      role,
		Status:    MEMBER_STATUS_ACTIVE,
		InvitedBy: "",
		JoinedAt:  &now,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// NewInvitedMembership creates a PENDING membership when a user is invited by an admin.
func NewInvitedMembership(id, userID, tenantID, inviterID string, role MemberRole) *Membership {
	now := time.Now()
	return &Membership{
		ID:        id,
		UserID:    userID,
		TenantID:  tenantID,
		Role:      role,
		Status:    MEMBER_STATUS_PENDING,
		InvitedBy: inviterID,
		JoinedAt:  nil,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Accept transitions a PENDING membership to ACTIVE and records joined_at.
// Returns ErrMembershipNotPending if the membership is not in PENDING state.
func (m *Membership) Accept() error {
	if m.Status != MEMBER_STATUS_PENDING {
		return ErrMembershipNotPending
	}
	now := time.Now()
	m.Status = MEMBER_STATUS_ACTIVE
	m.JoinedAt = &now
	m.UpdatedAt = now
	return nil
}

// Revoke transitions an ACTIVE membership to REVOKED.
// Returns ErrMembershipAlreadyRevoked if already revoked.
func (m *Membership) Revoke() error {
	if m.Status == MEMBER_STATUS_REVOKED {
		return ErrMembershipAlreadyRevoked
	}
	m.Status = MEMBER_STATUS_REVOKED
	m.UpdatedAt = time.Now()
	return nil
}

// IsActive reports whether this membership grants access.
func (m *Membership) IsActive() bool {
	return m.Status == MEMBER_STATUS_ACTIVE
}

// IsAdmin reports whether this membership grants administrative privileges.
// Both ROLE_OWNER and ROLE_ADMIN are considered administrators.
func (m *Membership) IsAdmin() bool {
	return m.Role == MEMBER_ROLE_OWNER || m.Role == MEMBER_ROLE_ADMIN
}

// IsOwner reports whether this membership is the founding member of the tenant.
func (m *Membership) IsOwner() bool {
	return m.Role == MEMBER_ROLE_OWNER
}
