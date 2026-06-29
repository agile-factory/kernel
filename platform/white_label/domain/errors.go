package domain

import (
	"net/http"

	"github.com/agile-factory/kernel/kernel/errors"
)

// ErrBrandNotFound is returned when a brand settings lookup yields no result.
var ErrBrandNotFound = &errors.AppError{
	Code:     "BRAND_NOT_FOUND",
	Message:  "brand settings not found",
	HTTPCode: http.StatusNotFound,
}

// ErrDomainNotPending is returned when verify-domain is called but the domain
// is not in PENDING or TIMEOUT status.
var ErrDomainNotPending = &errors.AppError{
	Code:     "DOMAIN_NOT_PENDING",
	Message:  "custom domain is not awaiting verification",
	HTTPCode: http.StatusConflict,
}

// ErrCustomDomainTaken is returned when a unique-constraint violation occurs
// on the custom_domain column (another tenant already verified this domain).
var ErrCustomDomainTaken = &errors.AppError{
	Code:     "CUSTOM_DOMAIN_TAKEN",
	Message:  "this custom domain is already in use by another tenant",
	HTTPCode: http.StatusConflict,
}
