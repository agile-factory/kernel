package domain

import (
	"context"
	"time"
)


// MembershipRepository defines the persistence contract for the Membership aggregate.
type MembershipRepository interface {
	// Save persists a new membership row.
	Save(ctx context.Context, m *Membership) error

	// FindByID retrieves a membership by its UUID.
	FindByID(ctx context.Context, id string) (*Membership, error)

	// FindByUserAndTenant retrieves the unique membership for a user/tenant pair.
	// Returns ErrMembershipNotFound when no row exists.
	FindByUserAndTenant(ctx context.Context, userID, tenantID string) (*Membership, error)

	// ListByTenant returns all memberships belonging to a tenant, ordered by created_at DESC.
	ListByTenant(ctx context.Context, tenantID string) ([]*Membership, error)

	// ListByUser returns all memberships a user holds across tenants.
	ListByUser(ctx context.Context, userID string) ([]*Membership, error)

	// UpdateStatus persists status and joined_at changes made by Accept() or Revoke().
	UpdateStatus(ctx context.Context, id string, status MemberStatus, joinedAt *time.Time) error
}
