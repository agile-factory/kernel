package usecase

import (
	"context"
	stderrors "errors"

	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/kernel/errors"
	auditdomain "github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/audit/domain"
	auditusecase "github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/audit/usecase"
	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/membership/domain"
	tenantdomain "github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/tenant/domain"
	"github.com/google/uuid"
)

// InviteMemberUseCase coordinates the invitation of a new member to a tenant.
type InviteMemberUseCase struct {
	repo       domain.MembershipRepository
	tenantRepo tenantdomain.TenantRepository
	audit      *auditusecase.LogEventUseCase
}

// NewInviteMemberUseCase creates a new InviteMemberUseCase.
func NewInviteMemberUseCase(repo domain.MembershipRepository, tenantRepo tenantdomain.TenantRepository, audit *auditusecase.LogEventUseCase) *InviteMemberUseCase {
	return &InviteMemberUseCase{
		repo:       repo,
		tenantRepo: tenantRepo,
		audit:      audit,
	}
}

// Execute invites a user (inviteeUserID) to a tenant on behalf of an admin (inviterID).
// The inviter must hold an ACTIVE ROLE_ADMIN membership in the target tenant.
func (uc *InviteMemberUseCase) Execute(ctx context.Context, inviterID, tenantID, inviteeUserID string, role domain.MemberRole) (*domain.Membership, error) {
	// Verify tenant exists
	tenant, err := uc.tenantRepo.FindByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if !tenant.IsActive {
		return nil, tenantdomain.ErrTenantNotFound
	}

	// Verify inviter is an active admin (ROLE_ADMIN or ROLE_OWNER) of this tenant.
	// "Not found" → 403 Forbidden (the caller has no membership in this tenant).
	// Any other error is an infrastructure failure and must NOT be masked as 403,
	// otherwise a transient DB outage would silently look like missing perms.
	inviterMembership, err := uc.repo.FindByUserAndTenant(ctx, inviterID, tenantID)
	if err != nil {
		if stderrors.Is(err, domain.ErrMembershipNotFound) {
			return nil, domain.ErrInsufficientMemberRole
		}
		return nil, errors.WrapInternal(err)
	}
	if !inviterMembership.IsActive() || !inviterMembership.IsAdmin() {
		return nil, domain.ErrInsufficientMemberRole
	}

	// ROLE_OWNER cannot be assigned via invitation — ownership is transferred separately
	if role == domain.MEMBER_ROLE_OWNER {
		return nil, domain.ErrCannotInviteAsOwner
	}

	// Check the invitee doesn't already have a membership.
	// Distinguish "not found" (expected, proceed) from infrastructure errors
	// (must propagate — silently treating them as "no membership exists" would
	// let us insert a duplicate row that only the UNIQUE constraint catches).
	existing, err := uc.repo.FindByUserAndTenant(ctx, inviteeUserID, tenantID)
	if err != nil && !stderrors.Is(err, domain.ErrMembershipNotFound) {
		return nil, errors.WrapInternal(err)
	}
	if existing != nil {
		return nil, domain.ErrMembershipAlreadyExists
	}

	// Build and persist the PENDING membership
	id := uuid.New().String()
	membership := domain.NewInvitedMembership(id, inviteeUserID, tenantID, inviterID, role)

	// The repo translates the UNIQUE(user_id, tenant_id) violation into
	// domain.ErrMembershipAlreadyExists; pass that through unwrapped so the
	// caller still sees a 409 instead of a generic 500.
	if err := uc.repo.Save(ctx, membership); err != nil {
		if stderrors.Is(err, domain.ErrMembershipAlreadyExists) {
			return nil, err
		}
		return nil, errors.WrapInternal(err)
	}

	uc.audit.Execute(ctx, auditusecase.LogEventParams{
		TenantID:   tenantID,
		ActorID:    inviterID,
		Action:     auditdomain.ACTION_MEMBER_INVITED,
		TargetType: "membership",
		TargetID:   membership.ID,
		Metadata: map[string]any{
			"invitee_id": inviteeUserID,
			"role":       role,
		},
	})

	return membership, nil
}
