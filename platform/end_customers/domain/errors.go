package domain

import (
	"net/http"

	"github.com/agile-factory/kernel/kernel/errors"
)

// ErrEndCustomerNotFound is returned when a lookup yields no result.
var ErrEndCustomerNotFound = &errors.AppError{
	Code:     "END_CUSTOMER_NOT_FOUND",
	Message:  "end customer not found",
	HTTPCode: http.StatusNotFound,
}

// ErrEndCustomerConflict is returned when a duplicate email or external_id
// already exists for the same tenant.
var ErrEndCustomerConflict = &errors.AppError{
	Code:     "END_CUSTOMER_CONFLICT",
	Message:  "an end customer with this email or external ID already exists",
	HTTPCode: http.StatusConflict,
}
