package domain

import (
	"strings"
	"time"
	"unicode"
)

// Tenant represents an isolated organizational unit in the multi-tenant architecture.
type Tenant struct {
	ID        string
	Name      string
	Slug      string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewTenant creates a new Tenant instance with the given name and slug.
func NewTenant(id, name, slug string) (*Tenant, error) {
	if strings.TrimSpace(name) == "" {
		return nil, ErrInvalidTenantName
	}

	normalizedSlug, err := NewSlug(slug)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	return &Tenant{
		ID:        id,
		Name:      strings.TrimSpace(name),
		Slug:      normalizedSlug,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// NewSlug validates and normalizes a slug value.
// A valid slug contains only lowercase letters, digits, and hyphens.
func NewSlug(raw string) (string, error) {
	slug := strings.ToLower(strings.TrimSpace(raw))

	if len(slug) < 2 || len(slug) > 100 {
		return "", ErrInvalidSlug
	}

	for _, ch := range slug {
		if !unicode.IsLower(ch) && !unicode.IsDigit(ch) && ch != '-' {
			return "", ErrInvalidSlug
		}
	}

	if strings.HasPrefix(slug, "-") || strings.HasSuffix(slug, "-") {
		return "", ErrInvalidSlug
	}

	return slug, nil
}
