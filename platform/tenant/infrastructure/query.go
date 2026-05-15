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

	// queryUpdateTenantName mutates the human-readable name.
	queryUpdateTenantName = `
		UPDATE tenants
		SET name = $2, updated_at = NOW()
		WHERE id = $1
	`

	// querySoftDeleteTenant flips is_active to FALSE.
	querySoftDeleteTenant = `
		UPDATE tenants
		SET is_active = FALSE, updated_at = NOW()
		WHERE id = $1
	`

	queryInsertTenantSettings = `
		INSERT INTO tenant_settings (
			tenant_id, logo_url, primary_color, locale, timezone, currency,
			data_region, custom_domain, payment_region, merchant_mode, b2b2c_enabled,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	queryGetTenantSettings = `
		SELECT
			tenant_id, logo_url, primary_color, locale, timezone, currency,
			data_region, custom_domain, payment_region, merchant_mode, b2b2c_enabled,
			created_at, updated_at
		FROM tenant_settings
		WHERE tenant_id = $1
	`

	queryUpdateTenantSettings = `
		UPDATE tenant_settings
		SET logo_url       = $2,
		    primary_color  = $3,
		    locale         = $4,
		    timezone       = $5,
		    currency       = $6,
		    data_region    = $7,
		    custom_domain  = $8,
		    payment_region = $9,
		    merchant_mode  = $10,
		    b2b2c_enabled  = $11,
		    updated_at     = NOW()
		WHERE tenant_id = $1
	`
)
