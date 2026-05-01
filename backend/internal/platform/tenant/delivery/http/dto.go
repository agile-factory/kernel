package http

import (
	"errors"
	"strings"
)

// CreateTenantRequest is the payload to create a new tenant.
type CreateTenantRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// Validate performs basic input checks on CreateTenantRequest.
func (r *CreateTenantRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(r.Slug) == "" {
		return errors.New("slug is required")
	}
	return nil
}

// TenantResponse is the public representation of a Tenant.
type TenantResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	IsActive  bool   `json:"isActive"`
	CreatedAt string `json:"createdAt"`
}
