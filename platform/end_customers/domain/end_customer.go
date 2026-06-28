// Package domain holds the end_customers platform's domain types and repository contract.
package domain

import (
	"time"

	apperrors "github.com/agile-factory/kernel/kernel/errors"
)


// EndCustomerStatus represents the lifecycle state of an end-customer.
type EndCustomerStatus string

const (
	StatusActive    EndCustomerStatus = "ACTIVE"
	StatusSuspended EndCustomerStatus = "SUSPENDED"
	StatusDeleted   EndCustomerStatus = "DELETED"
)

// EndCustomer is a client final managed by a tenant in B2B2C context.
// Distinct from auth.User (workspace members): end-customers access the
// tenant's customer-facing portal, never the back-office.
type EndCustomer struct {
	ID         string
	TenantID   string
	ExternalID string            // optional business identifier (tenant-assigned customer number)
	Email      string            // nullable — CEDEAO customers may have phone only
	Phone      string            // E.164 format (+221XXXXXXXXX)
	FirstName  string
	LastName   string
	Status     EndCustomerStatus
	Tags       []string
	Metadata   map[string]string // free-form fields, max 20 keys × 256 chars (enforced in use cases)
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  *time.Time
}

// FullName returns the display name of the end-customer.
func (ec *EndCustomer) FullName() string {
	return ec.FirstName + " " + ec.LastName
}

// Suspend transitions the end-customer from ACTIVE to SUSPENDED.
// Returns an error if the current status is not ACTIVE.
func (ec *EndCustomer) Suspend() error {
	if ec.Status != StatusActive {
		return apperrors.BadRequest("end_customer must be ACTIVE to be suspended")
	}
	ec.Status = StatusSuspended
	return nil
}

// Restore transitions the end-customer from SUSPENDED back to ACTIVE.
// Returns an error if the current status is not SUSPENDED.
func (ec *EndCustomer) Restore() error {
	if ec.Status != StatusSuspended {
		return apperrors.BadRequest("end_customer must be SUSPENDED to be restored")
	}
	ec.Status = StatusActive
	return nil
}
