package domain

import "time"

// SupportedTemplateKeys enumerates the built-in notification events.
var SupportedTemplateKeys = map[string]bool{
	"welcome":            true,
	"invoice_paid":       true,
	"payment_reminder":   true,
	"otp_reset":          true,
}

// CustomerTemplate is a per-tenant override of a default template.
type CustomerTemplate struct {
	ID          string
	TenantID    string
	Key         string  // "welcome" | "invoice_paid" | "payment_reminder" | "otp_reset"
	Channel     Channel
	Locale      string  // "fr" | "en" | "wo"
	SubjectTmpl string  // text/template string for email subject
	BodyTmpl    string  // text/template string (HTML for email, plain text for SMS)
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TemplateData is the context passed to text/template rendering.
type TemplateData struct {
	CustomerFirstName string
	CustomerLastName  string
	TenantAppName     string
	TenantLogoURL     string
	TenantPrimaryColor string
	SupportEmail      string
	InvoiceAmount     string
	InvoiceCurrency   string
	OTPCode           string
	ResetLink         string
}
