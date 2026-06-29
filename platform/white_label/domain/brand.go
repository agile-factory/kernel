// Package domain holds the white_label platform's domain types and repository contract.
package domain

import "time"

// DomainVerificationStatus represents the DNS verification lifecycle for a custom domain.
type DomainVerificationStatus string

const (
	// DomainNotSet means no custom domain has been configured.
	DomainNotSet DomainVerificationStatus = "NOT_SET"
	// DomainPending means a domain was submitted and awaits DNS verification.
	DomainPending DomainVerificationStatus = "PENDING"
	// DomainTimeout means the DNS lookup timed out — retry is safe.
	DomainTimeout DomainVerificationStatus = "TIMEOUT"
	// DomainVerified means the TXT record was found and matched the stored token.
	DomainVerified DomainVerificationStatus = "VERIFIED"
	// DomainFailed means the TXT record was found but did not match — reconfigure.
	DomainFailed DomainVerificationStatus = "FAILED"
)

// BrandSettings holds all per-tenant branding configuration.
// There is at most one row per tenant (upsert pattern keyed by TenantID).
//
// EmailFooterHTML is used for server-side email rendering only.
// It MUST NOT be rendered in a browser without sanitisation (XSS risk).
type BrandSettings struct {
	TenantID string

	// App identity
	AppName    string // max 50 chars
	LogoURL    string // must be https://
	FaviconURL string // must be https://

	// Colour system
	PrimaryColor   string // hex #RRGGBB
	SecondaryColor string // hex #RRGGBB
	AccentColor    string // hex #RRGGBB

	// Typography
	FontFamily    string // inter | roboto | lato | custom
	CustomFontURL string // required when FontFamily == "custom", must be https://

	// Custom domain (DNS CNAME verification)
	CustomDomain           string
	CustomDomainStatus     DomainVerificationStatus
	CustomDomainToken      string     // "_kernel-verify=<uuid>" — generated, immutable after PENDING
	CustomDomainVerifiedAt *time.Time // set when status transitions to VERIFIED

	// Email template overrides (RFC-0036)
	// EmailFooterHTML must only be used for server-side email rendering.
	EmailFooterHTML string // max 2000 chars
	EmailLogoURL    string // must be https://

	// Legal & support
	TermsURL     string // must be https://
	PrivacyURL   string // must be https://
	SupportEmail string
	SupportPhone string

	UpdatedAt time.Time
}
