package permission

import (
	"context"

	kernelauth "github.com/agile-factory/kernel/kernel/auth"
	"github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/platform/membership/domain"
)

// Checker is the port that use-cases call to assert a capability.
// Call Require to gate an action; it returns nil when the caller has the
// permission and a *errors.AppError (HTTP 403) otherwise.
type Checker interface {
	// Require asserts that the authenticated principal in ctx holds perm for
	// the tenant in ctx. Returns nil on success, errors.PermissionDenied on
	// failure, or an infrastructure error if the membership lookup fails.
	Require(ctx context.Context, perm Permission) error
}

// MembershipBasedChecker is the v1 implementation of Checker. It reads the
// caller's role from the membership table (via FindByUserAndTenant) and
// delegates to the in-memory role→permission map. No extra SQL tables needed.
type MembershipBasedChecker struct {
	repo domain.MembershipRepository
}

// New returns a Checker backed by the provided membership repository.
// Inject this into use-cases that need permission checks.
func New(repo domain.MembershipRepository) Checker {
	return &MembershipBasedChecker{repo: repo}
}

// Require implements Checker.
//
// Flow:
//  1. Platform super-admins (platform_role == "ROLE_ADMIN" from JWT) bypass all checks.
//  2. userID and tenantID are read from context (set by delivery/http middleware).
//  3. The membership is fetched; only ACTIVE memberships are accepted.
//  4. The role→permission map determines whether perm is granted.
func (c *MembershipBasedChecker) Require(ctx context.Context, perm Permission) error {
	// Platform super-admin bypass (internal tooling / support)
	if kernelauth.IsPlatformAdmin(ctx) {
		return nil
	}

	userID := kernelauth.UserIDFromContext(ctx)
	tenantID := kernelauth.TenantIDFromContext(ctx)

	if userID == "" || tenantID == "" {
		return errors.Unauthorized("missing identity in context")
	}

	m, err := c.repo.FindByUserAndTenant(ctx, userID, tenantID)
	if err != nil {
		if appErr, ok := err.(*errors.AppError); ok {
			return appErr
		}
		return errors.WrapInternal(err)
	}

	if !m.IsActive() {
		return errors.Forbidden("membership is not active")
	}

	if !roleHasPermission(m.Role, perm) {
		return errors.PermissionDenied("insufficient role for " + string(perm))
	}

	return nil
}
