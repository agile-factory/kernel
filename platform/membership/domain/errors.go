package domain

import "github.com/agile-factory/kernel/kernel/errors"

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

	// ErrInsufficientMemberRole is returned when the acting user lacks ROLE_ADMIN or ROLE_OWNER in the tenant.
	ErrInsufficientMemberRole = errors.Forbidden("Only tenant admins can perform this action")

	// ErrCannotRevokeOwner is returned when someone attempts to revoke a ROLE_OWNER membership.
	// Ownership must be explicitly transferred before the owner's membership can be changed.
	ErrCannotRevokeOwner = errors.Forbidden("The tenant owner cannot be revoked; transfer ownership first")

	// ErrCannotInviteAsOwner is returned when an invitation specifies ROLE_OWNER as the target role.
	// Ownership is established at tenant creation and transferred via a dedicated endpoint, never via invitation.
	ErrCannotInviteAsOwner = errors.BadRequest("ROLE_OWNER cannot be assigned through an invitation")

	// ErrCannotRevokeOwnMembership is returned when an admin tries to revoke themselves.
	ErrCannotRevokeOwnMembership = errors.BadRequest("You cannot revoke your own membership")

	// ErrInviteTokenNotFound is returned for an unknown invite token.
	ErrInviteTokenNotFound = errors.NotFound("Invite token not found")

	// ErrInviteTokenExpired is returned when the invite is past expires_at.
	ErrInviteTokenExpired = errors.BadRequest("Invite token has expired")

	// ErrInviteTokenAlreadyConsumed is returned on replay of a redeemed invite.
	ErrInviteTokenAlreadyConsumed = errors.Conflict("Invite token has already been used")

	// ErrInviteTokenRevoked is returned when an admin has revoked the invite.
	ErrInviteTokenRevoked = errors.Forbidden("Invite token has been revoked")

	// ErrInviteEmailMismatch is returned when the redeeming caller's email does
	// not match the email the invite was issued to.
	ErrInviteEmailMismatch = errors.Forbidden("This invite was issued for a different email address")
)
