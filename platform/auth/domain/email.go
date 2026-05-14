package domain

import (
	"strings"

	"github.com/AgileFactory/kernel/kernel/validator"
)

// Email is a value object representing an email address.
type Email struct {
	Value string
}

// NewEmail creates a new valid Email value object or returns an error.
func NewEmail(val string) (Email, error) {
	val = strings.TrimSpace(val)
	val = strings.ToLower(val)

	if !validator.IsValidEmail(val) {
		return Email{}, ErrInvalidEmail
	}

	return Email{Value: val}, nil
}

// String implements the Stringer interface.
func (e Email) String() string {
	return e.Value
}
