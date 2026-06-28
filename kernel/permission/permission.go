// Package permission provides a lightweight role-based permission checker
// for the AgileFactory kernel. In v1 it operates entirely from the existing
// membership.Role field — no new SQL tables required (YAGNI; see ADR-0012).
// Custom role tables will be added in v2 when a consumer SaaS needs them (see ADR-0011).
package permission

// Permission is a dot-separated capability string of the form "module:action".
// Each platform module declares its own constants in this package so that
// consumer repos have a single import for all capability checks.
type Permission string

// Tenant permissions.
const (
	TenantView   Permission = "tenant:view"
	TenantUpdate Permission = "tenant:update"
	TenantDelete Permission = "tenant:delete"
)

// Membership permissions.
const (
	MemberInvite Permission = "member:invite"
	MemberRevoke Permission = "member:revoke"
	MemberList   Permission = "member:list"
)

// Audit permissions.
const (
	AuditView Permission = "audit:view"
)

// User profile permissions.
const (
	UserProfileView      Permission = "user:profile:view"
	UserProfileUpdateOwn Permission = "user:profile:update_own"
)

// Storage permissions.
const (
	StorageUpload Permission = "storage:upload"
	StorageDelete Permission = "storage:delete"
)

// EndCustomer permissions (platform/end_customers — RFC-0033).
const (
	EndCustomerCreate  Permission = "end_customer:create"
	EndCustomerList    Permission = "end_customer:list"
	EndCustomerUpdate  Permission = "end_customer:update"
	EndCustomerDelete  Permission = "end_customer:delete"
	EndCustomerSuspend Permission = "end_customer:suspend"
)

// Brand permissions (platform/white_label — RFC-0035).
const (
	BrandRead  Permission = "brand:read"
	BrandWrite Permission = "brand:write"
)

const (
	CustomerNotificationView  Permission = "customer_notification:view"
	CustomerTemplateWrite     Permission = "customer_template:write"
)
