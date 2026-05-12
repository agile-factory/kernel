package usecase

import (
	"context"

	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/kernel/errors"
	auditdomain "github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/audit/domain"
	auditusecase "github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/audit/usecase"
	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/membership/domain"
)

// RevokeMemberUseCase transitions a membership to REVOKED.
type RevokeMemberUseCase struct {
	repo  domain.MembershipRepository
	audit *auditusecase.LogEventUseCase
}

// NewRevokeMemberUseCase creates a new RevokeMemberUseCase.
func NewRevokeMemberUseCase(repo domain.MembershipRepository, audit *auditusecase.LogEventUseCase) *RevokeMemberUseCase {
	return &RevokeMemberUseCase{
		repo:  repo,
		audit: audit,
	}
}

// Execute revokes a membership.
// The revoker must be an active ROLE_ADMIN in the same tenant, and cannot revoke themselves.
func (uc *RevokeMemberUseCase) Execute(ctx context.Context, membershipID, revokerID string) (*domain.Membership, error) {
	// Load the target membership
	membership, err := uc.repo.FindByID(ctx, membershipID)
	if err != nil {
		return nil, err
	}

	// Self-revocation guard
	if membership.UserID == revokerID {
		return nil, domain.ErrCannotRevokeOwnMembership
	}

	// Verify revoker is an active admin (ROLE_ADMIN or ROLE_OWNER) in the same tenant
	revokerMembership, err := uc.repo.FindByUserAndTenant(ctx, revokerID, membership.TenantID)
	if err != nil || !revokerMembership.IsActive() || !revokerMembership.IsAdmin() {
		return nil, domain.ErrInsufficientMemberRole
	}

	// Owners cannot be revoked — they must transfer ownership first
	if membership.IsOwner() {
		return nil, domain.ErrCannotRevokeOwner
	}

	// State-machine transition (validates not-already-revoked internally)
	if err := membership.Revoke(); err != nil {
		return nil, err
	}

	// Persist — joined_at is untouched on revocation (pass nil)
	if err := uc.repo.UpdateStatus(ctx, membership.ID, membership.Status, nil); err != nil {
		return nil, errors.WrapInternal(err)
	}

	uc.audit.Execute(ctx, auditusecase.LogEventParams{
		TenantID:   membership.TenantID,
		ActorID:    revokerID,
		Action:     auditdomain.ACTION_MEMBER_REVOKED,
		TargetType: "membership",
		TargetID:   membership.ID,
		Metadata: map[string]any{
			"revoked_user_id": membership.UserID,
		},
	})

	return membership, nil
}
