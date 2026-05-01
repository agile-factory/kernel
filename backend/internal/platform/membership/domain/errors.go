package domain

import "github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/kernel/errors"

// Sentinel errors for the Membership domain.
var (
	// ErrMembershipNotFound is returned when no membership row matches the query.
	ErrMembershipNotFound = errors.NotFound("Membership not found")

	// ErrMembershipAlreadyExists is returned when a user is invited to a tenant
	// they already have a membership in (enforced by UNIQUE constraint).
	ErrMembershipAlreadyExists = errors.Conflict("User already has a membership in this tenant")

	// ErrMembershipNotPending is returned when Accept() is called on a non-PENDING membership.
	ErrMembershipNotPending = errors.BadRequest("Membership is not in PENDING state")

	// ErrMembershipAlreadyRevoked is returned when Revoke() is called on an already-revoked membership.
	ErrMembershipAlreadyRevoked = errors.BadRequest("Membership is already revoked")

	// ErrInsufficientMemberRole is returned when the acting user lacks ROLE_ADMIN in the tenant.
	ErrInsufficientMemberRole = errors.Forbidden("Only tenant admins can perform this action")

	// ErrCannotRevokeOwnMembership is returned when an admin tries to revoke themselves.
	ErrCannotRevokeOwnMembership = errors.BadRequest("You cannot revoke your own membership")
)
