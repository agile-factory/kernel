// Package domain holds the user platform's domain types and repository contract.
package domain

import (
	"context"
	"strings"
	"time"
)

// UserProfile is the public-facing profile of a user within the platform.
// It is distinct from auth.User (identity + credentials) — see RFC-0009.
type UserProfile struct {
	UserID      string
	DisplayName string
	AvatarURL   string
	Bio         string
	Locale      string    // e.g. "fr-SN", "en-US"
	Timezone    string    // e.g. "Africa/Dakar", "UTC"
	UpdatedAt   time.Time
}

// NewFromEmail creates a minimal profile with DisplayName inferred from the
// email prefix. Used for lazy initialisation on first profile access.
func NewFromEmail(userID, email string) *UserProfile {
	name := email
	if at := strings.Index(email, "@"); at > 0 {
		name = email[:at]
	}
	return &UserProfile{
		UserID:      userID,
		DisplayName: name,
		Locale:      "fr-SN",
		Timezone:    "Africa/Dakar",
		UpdatedAt:   time.Now().UTC(),
	}
}

// UserProfileRepository is the persistence contract for UserProfile.
type UserProfileRepository interface {
	// FindByUserID returns the profile for userID, or errors.NotFound if absent.
	FindByUserID(ctx context.Context, userID string) (*UserProfile, error)
	// Upsert inserts or updates the profile row.
	Upsert(ctx context.Context, profile *UserProfile) error
}
