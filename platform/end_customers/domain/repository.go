package domain

import "context"

// ListParams carries the filters and cursor for listing end-customers.
type ListParams struct {
	TenantID string
	Cursor   string             // opaque cursor from a previous response
	Limit    int                // max items per page; defaults to 20 in use case
	Status   *EndCustomerStatus // nil = all non-deleted
	Tag      string             // filter by tag (exact match)
	Search   string             // full-text search on first_name + last_name + email
}

// EndCustomerRepository is the persistence contract for EndCustomer.
// Implementations live in infrastructure/; the domain never imports them.
type EndCustomerRepository interface {
	// Create inserts a new end-customer row.
	Create(ctx context.Context, ec *EndCustomer) error

	// FindByID returns the end-customer for the given tenant + id.
	// Returns ErrEndCustomerNotFound when absent or soft-deleted.
	FindByID(ctx context.Context, tenantID, id string) (*EndCustomer, error)

	// FindByEmail returns the end-customer matching email in tenant.
	// Returns ErrEndCustomerNotFound when absent.
	FindByEmail(ctx context.Context, tenantID, email string) (*EndCustomer, error)

	// FindByExternalID returns the end-customer matching external_id in tenant.
	// Returns ErrEndCustomerNotFound when absent.
	FindByExternalID(ctx context.Context, tenantID, externalID string) (*EndCustomer, error)

	// Update persists all mutable fields of ec (status, names, email, phone,
	// external_id, metadata, updated_at).
	Update(ctx context.Context, ec *EndCustomer) error

	// SoftDelete sets deleted_at = NOW() and status = DELETED.
	SoftDelete(ctx context.Context, tenantID, id string) error

	// List returns a page of end-customers. Returns the next cursor (empty
	// string when there are no more pages) as the second value.
	List(ctx context.Context, params ListParams) ([]*EndCustomer, string, error)

	// Count returns the total number of non-deleted end-customers for the tenant.
	// Optionally filtered by status.
	Count(ctx context.Context, tenantID string, status *EndCustomerStatus) (int64, error)
}
