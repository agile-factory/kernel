package domain

import "context"

// TenantRepository defines the persistence contract for the Tenant aggregate.
type TenantRepository interface {
	// Save persists a new tenant to the database.
	Save(ctx context.Context, t *Tenant) error

	// FindByID retrieves a tenant by its unique identifier.
	FindByID(ctx context.Context, id string) (*Tenant, error)

	// FindBySlug retrieves a tenant by its unique URL-safe slug.
	FindBySlug(ctx context.Context, slug string) (*Tenant, error)

	// ExistsBySlug checks if a tenant with the given slug already exists.
	ExistsBySlug(ctx context.Context, slug string) (bool, error)

	// UpdateName changes the human-readable tenant name; slug is immutable.
	UpdateName(ctx context.Context, id, name string) error

	// SoftDelete flips is_active to FALSE. The row stays in the table so
	// references (audit logs, memberships) remain intact. Hard-delete is a
	// separate GDPR flow (RFC-0027, super-admin only).
	SoftDelete(ctx context.Context, id string) error
}
