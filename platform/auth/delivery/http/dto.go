package http

import "errors"

// ─── Signup Owner ────────────────────────────────────────────────────────────

// SignupOwnerRequest is the payload for POST /api/v1/auth/signup-owner.
type SignupOwnerRequest struct {
	TenantName string `json:"tenantName"`
	TenantSlug string `json:"tenantSlug"`
	Email      string `json:"email"`
	Password   string `json:"password"`
}

func (r *SignupOwnerRequest) Validate() error {
	if r.TenantName == "" {
		return errors.New("tenantName is required")
	}
	if r.TenantSlug == "" {
		return errors.New("tenantSlug is required")
	}
	if r.Email == "" {
		return errors.New("email is required")
	}
	if len(r.Password) < 12 {
		return errors.New("password must be at least 12 characters")
	}
	return nil
}

// SignupOwnerResponse bundles the just-created tenant/user identifiers and
// the immediately usable access + refresh tokens.
type SignupOwnerResponse struct {
	TenantID                 string  `json:"tenantId"`
	UserID                   string  `json:"userId"`
	MembershipID             string  `json:"membershipId"`
	AccessToken              string  `json:"accessToken"`
	RefreshToken             string  `json:"refreshToken"`
	EmailVerificationPending bool    `json:"emailVerificationPending"`
	User                     UserDTO `json:"user"`
}

// ─── Accept Invite ───────────────────────────────────────────────────────────

// AcceptInviteRequest is the payload for POST /api/v1/auth/accept-invite.
// Password is only required when the email is unknown to the system.
type AcceptInviteRequest struct {
	Token    string `json:"token"`
	Password string `json:"password,omitempty"`
}

func (r *AcceptInviteRequest) Validate() error {
	if r.Token == "" {
		return errors.New("token is required")
	}
	if r.Password != "" && len(r.Password) < 12 {
		return errors.New("password must be at least 12 characters")
	}
	return nil
}

// AcceptInviteResponse mirrors SignupOwner but for invite redemption.
type AcceptInviteResponse struct {
	UserID       string  `json:"userId"`
	MembershipID string  `json:"membershipId"`
	TenantID     string  `json:"tenantId"`
	Role         string  `json:"role"`
	AccessToken  string  `json:"accessToken"`
	RefreshToken string  `json:"refreshToken"`
	User         UserDTO `json:"user"`
}

// ─── Verify Email ────────────────────────────────────────────────────────────

// VerifyEmailRequest is the payload for POST /api/v1/auth/verify-email.
type VerifyEmailRequest struct {
	Token string `json:"token"`
}

func (r *VerifyEmailRequest) Validate() error {
	if r.Token == "" {
		return errors.New("token is required")
	}
	return nil
}

// ─── Login ───────────────────────────────────────────────────────────────────

// LoginRequest is the payload for POST /api/v1/auth/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r *LoginRequest) Validate() error {
	if r.Email == "" {
		return errors.New("email is required")
	}
	if r.Password == "" {
		return errors.New("password is required")
	}
	return nil
}

// LoginResponse is what the login endpoint returns.
type LoginResponse struct {
	AccessToken  string  `json:"accessToken"`
	RefreshToken string  `json:"refreshToken"`
	User         UserDTO `json:"user"`
}

// ─── Refresh / Logout ────────────────────────────────────────────────────────

// RefreshRequest is the payload for POST /api/v1/auth/refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

func (r *RefreshRequest) Validate() error {
	if r.RefreshToken == "" {
		return errors.New("refreshToken is required")
	}
	return nil
}

// RefreshResponse is the rotation result.
type RefreshResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

// LogoutRequest is the payload for POST /api/v1/auth/logout. RefreshToken is
// optional — when omitted, the endpoint is a no-op (idempotent).
type LogoutRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// ─── Shared ──────────────────────────────────────────────────────────────────

// UserDTO represents public user data returned by the auth endpoints.
type UserDTO struct {
	ID            string              `json:"id"`
	Email         string              `json:"email"`
	Role          string              `json:"role"`
	EmailVerified bool                `json:"emailVerified"`
	Status        string              `json:"status"`
	Memberships   []MembershipSummary `json:"memberships,omitempty"`
}

// MembershipSummary is a compact view of a membership embedded in auth responses.
type MembershipSummary struct {
	MembershipID string `json:"membershipId"`
	TenantID     string `json:"tenantId"`
	Role         string `json:"role"`
	Status       string `json:"status"`
}
