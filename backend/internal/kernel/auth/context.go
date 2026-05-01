// Package auth provides context keys and helpers for propagating
// authentication and tenant information through the HTTP request context.
package auth

import "context"

// contextKey is an unexported type for context keys in this package.
// Using a typed key prevents collisions with keys from other packages.
type contextKey string

const (
	// contextKeyUserID is the key under which the authenticated user's ID is stored.
	contextKeyUserID contextKey = "user_id"

	// contextKeyTenantID is the key under which the active tenant ID is stored.
	contextKeyTenantID contextKey = "tenant_id"

	// contextKeyPlatformRole is the key under which the platform-level role from the JWT is stored.
	// This is distinct from the tenant-level role stored in the memberships table.
	// A user with platform ROLE_ADMIN has super-admin privileges across all tenants.
	contextKeyPlatformRole contextKey = "platform_role"
)

// WithUserID returns a new context carrying the given userID.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, contextKeyUserID, userID)
}

// UserIDFromContext extracts the authenticated user ID from the context.
// Returns an empty string if no user ID is present.
func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(contextKeyUserID).(string)
	return v
}

// WithTenantID returns a new context carrying the given tenantID.
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, contextKeyTenantID, tenantID)
}

// TenantIDFromContext extracts the active tenant ID from the context.
// Returns an empty string if no tenant ID is present.
func TenantIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(contextKeyTenantID).(string)
	return v
}

// WithPlatformRole returns a new context carrying the platform-level role from the JWT.
func WithPlatformRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, contextKeyPlatformRole, role)
}

// PlatformRoleFromContext extracts the platform-level role from the context.
// Returns an empty string if not present.
func PlatformRoleFromContext(ctx context.Context) string {
	v, _ := ctx.Value(contextKeyPlatformRole).(string)
	return v
}

// IsPlatformAdmin reports whether the caller holds the platform-level ROLE_ADMIN.
// This grants super-admin access across all tenants.
func IsPlatformAdmin(ctx context.Context) bool {
	return PlatformRoleFromContext(ctx) == "ROLE_ADMIN"
}
