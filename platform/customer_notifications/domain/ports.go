package domain

import "context"

// EmailSender sends transactional emails.
// Shared port with platform/notification (RFC-0019).
type EmailSender interface {
	Send(ctx context.Context, to, subject, htmlBody string) error
}

// SMSSender sends SMS messages. Returns the provider's external ID for tracking.
type SMSSender interface {
	Send(ctx context.Context, to, body string) (externalID string, err error)
}

// EndCustomerContact holds the contact details of an end-customer.
type EndCustomerContact struct {
	Email     string
	Phone     string // E.164 format, may be empty
	FirstName string
	LastName  string
}

// EndCustomerResolver looks up end-customer contact information.
// Implemented in infrastructure via a SQL query against end_customers.
type EndCustomerResolver interface {
	GetContact(ctx context.Context, endCustomerID, tenantID string) (*EndCustomerContact, error)
}

// BrandData captures the per-tenant branding needed for notification templates.
type BrandData struct {
	AppName      string
	LogoURL      string
	PrimaryColor string
	SupportEmail string
}

// BrandResolver looks up a tenant's brand settings for template rendering.
// Implemented in infrastructure via a SQL query against brand_settings.
type BrandResolver interface {
	GetBrand(ctx context.Context, tenantID string) (*BrandData, error)
}
