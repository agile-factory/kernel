package domain

import "github.com/agile-factory/kernel/kernel/errors"

// Sentinel errors for the Tenant domain.
var (
	// ErrTenantNotFound is returned when a tenant cannot be located by ID or slug.
	ErrTenantNotFound = errors.TenantNotFound()

	// ErrTenantAlreadyExists is returned when a tenant with the same slug already exists.
	ErrTenantAlreadyExists = errors.Conflict("A tenant with this slug already exists")

	// ErrInvalidSlug is returned when the slug contains invalid characters or length.
	ErrInvalidSlug = errors.BadRequest("Slug must be 2-100 chars, lowercase letters, digits, and hyphens only")

	// ErrInvalidTenantName is returned when the tenant name is blank.
	ErrInvalidTenantName = errors.BadRequest("Tenant name must not be empty")

	// ErrTenantSettingsNotFound is returned when no tenant_settings row exists
	// for a tenant. New tenants get one inserted by the create flow, so this
	// can only happen for legacy tenants that predate the 0002 migration.
	ErrTenantSettingsNotFound = errors.NotFound("Tenant settings not found")
)

