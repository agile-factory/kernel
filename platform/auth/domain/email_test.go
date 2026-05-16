package domain_test

import (
	"testing"

	"github.com/agile-factory/kernel/platform/auth/domain"
)

// ─── NewEmail ────────────────────────────────────────────────────────────────

func TestNewEmail_ValidEmails(t *testing.T) {
	valid := []string{
		"user@example.com",
		"USER@EXAMPLE.COM",           // doit être normalisé en lowercase
		"  user@example.com  ",       // espaces doivent être trimmés
		"user+tag@sub.domain.io",
		"a@b.co",
	}

	for _, raw := range valid {
		t.Run(raw, func(t *testing.T) {
			email, err := domain.NewEmail(raw)
			if err != nil {
				t.Fatalf("NewEmail(%q) returned unexpected error: %v", raw, err)
			}
			if email.Value == "" {
				t.Error("Email.Value should not be empty for a valid input")
			}
		})
	}
}

func TestNewEmail_NormalizesLowercase(t *testing.T) {
	email, err := domain.NewEmail("User@Example.COM")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if email.Value != "user@example.com" {
		t.Errorf("expected lowercase email, got %q", email.Value)
	}
}

func TestNewEmail_TrimSpaces(t *testing.T) {
	email, err := domain.NewEmail("  hello@world.com  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if email.Value != "hello@world.com" {
		t.Errorf("expected trimmed email, got %q", email.Value)
	}
}

func TestNewEmail_InvalidEmails(t *testing.T) {
	invalid := []string{
		"",
		"notanemail",
		"@nodomain.com",
		"noatsign.com",
		"double@@domain.com",
		"user@",
		"   ",
	}

	for _, raw := range invalid {
		t.Run(raw, func(t *testing.T) {
			_, err := domain.NewEmail(raw)
			if err == nil {
				t.Errorf("NewEmail(%q) should have returned ErrInvalidEmail", raw)
			}
		})
	}
}

func TestEmail_String(t *testing.T) {
	email, _ := domain.NewEmail("test@agile.io")
	if email.String() != "test@agile.io" {
		t.Errorf("String(): got %q, want %q", email.String(), "test@agile.io")
	}
}

// ─── ErrInvalidEmail sentinel ────────────────────────────────────────────────

func TestNewEmail_ReturnsErrInvalidEmail(t *testing.T) {
	_, err := domain.NewEmail("bad-email")
	if err != domain.ErrInvalidEmail {
		t.Errorf("expected ErrInvalidEmail, got %v", err)
	}
}
