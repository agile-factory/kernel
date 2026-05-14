package usecase

import (
	"context"

	"github.com/AgileFactory/kernel/kernel/errors"
	auditdomain "github.com/AgileFactory/kernel/platform/audit/domain"
	auditusecase "github.com/AgileFactory/kernel/platform/audit/usecase"
	"github.com/AgileFactory/kernel/platform/membership/domain"
)

// AcceptInviteUseCase coordinates a user accepting an invitation.
type AcceptInviteUseCase struct {
	repo  domain.MembershipRepository
	audit *auditusecase.LogEventUseCase
}

// NewAcceptInviteUseCase creates a new AcceptInviteUseCase.
func NewAcceptInviteUseCase(repo domain.MembershipRepository, audit *auditusecase.LogEventUseCase) *AcceptInviteUseCase {
	return &AcceptInviteUseCase{
		repo:  repo,
		audit: audit,
	}
}

// Execute accepts a pending invitation.
// callerUserID must match the membership's UserID to prevent cross-user acceptance.
func (uc *AcceptInviteUseCase) Execute(ctx context.Context, membershipID, callerUserID string) (*domain.Membership, error) {
	membership, err := uc.repo.FindByID(ctx, membershipID)
	if err != nil {
		return nil, err
	}

	// Ownership check — only the invited user can accept their own invitation
	if membership.UserID != callerUserID {
		return nil, domain.ErrInsufficientMemberRole
	}

	// State-machine transition (validates PENDING state internally)
	if err := membership.Accept(); err != nil {
		return nil, err
	}

	// Persist the status + joined_at change
	if err := uc.repo.UpdateStatus(ctx, membership.ID, membership.Status, membership.JoinedAt); err != nil {
		return nil, errors.WrapInternal(err)
	}

	uc.audit.Execute(ctx, auditusecase.LogEventParams{
		TenantID:   membership.TenantID,
		ActorID:    callerUserID,
		Action:     auditdomain.ACTION_MEMBER_ACCEPTED,
		TargetType: "membership",
		TargetID:   membership.ID,
		Metadata: map[string]any{
			"role": membership.Role,
		},
	})

	return membership, nil
}
