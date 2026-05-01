package infrastructure

const (
	// queryInsertTenant inserts a new tenant row.
	queryInsertTenant = `
		INSERT INTO tenants (id, name, slug, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	// queryGetTenantByID retrieves a tenant by primary key.
	queryGetTenantByID = `
		SELECT id, name, slug, is_active, created_at, updated_at
		FROM tenants
		WHERE id = $1
	`

	// queryGetTenantBySlug retrieves a tenant by its unique slug.
	queryGetTenantBySlug = `
		SELECT id, name, slug, is_active, created_at, updated_at
		FROM tenants
		WHERE slug = $1
	`

	// queryTenantExistsBySlug checks slug uniqueness efficiently.
	queryTenantExistsBySlug = `
		SELECT EXISTS (
			SELECT 1 FROM tenants WHERE slug = $1
		)
	`
)
