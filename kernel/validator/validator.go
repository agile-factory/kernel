// Package validator hosts small, stateless input checks shared across the
// platform packages (email shape, non-empty strings, …). It is intentionally
// minimal: anything more complex than a regex belongs in a domain validator
// next to the type it constrains, not here.
package validator

import (
	"regexp"
)

// Simple email regex for basic validation
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// IsValidEmail returns true if the email format is basically valid.
func IsValidEmail(email string) bool {
	if len(email) < 3 || len(email) > 254 {
		return false
	}
	return emailRegex.MatchString(email)
}

// IsNotEmpty checks if a string is not empty or just whitespaces.
func IsNotEmpty(s string) bool {
	for _, c := range s {
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			return true
		}
	}
	return false
}
