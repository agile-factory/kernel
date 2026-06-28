package domain

import (
	"net/http"

	"github.com/agile-factory/kernel/kernel/errors"
)

var (
	// ErrTemplateNotFound is returned when no template override exists for the given key.
	ErrTemplateNotFound = &errors.AppError{
		Code:     "TEMPLATE_NOT_FOUND",
		Message:  "notification template not found",
		HTTPCode: http.StatusNotFound,
	}

	// ErrTemplateInvalid is returned when the provided template fails text/template parsing.
	ErrTemplateInvalid = &errors.AppError{
		Code:     "TEMPLATE_INVALID",
		Message:  "notification template has invalid syntax",
		HTTPCode: http.StatusBadRequest,
	}

	// ErrUnsupportedTemplateKey is returned when the key is not in SupportedTemplateKeys.
	ErrUnsupportedTemplateKey = &errors.AppError{
		Code:     "UNSUPPORTED_TEMPLATE_KEY",
		Message:  "unsupported notification template key",
		HTTPCode: http.StatusBadRequest,
	}

	// ErrEndCustomerNotFound is returned when the end-customer cannot be resolved.
	ErrEndCustomerNotFound = &errors.AppError{
		Code:     "END_CUSTOMER_NOT_FOUND",
		Message:  "end-customer not found",
		HTTPCode: http.StatusNotFound,
	}
)
