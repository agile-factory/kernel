package usecase

import (
	"context"

	"github.com/AgileFactory/kernel/platform/membership/domain"
)

// ListMembersUseCase retrieves all memberships for a given tenant or user.
type ListMembersUseCase struct {
	membershipRepo domain.MembershipRepository
}

// NewListMembersUseCase creates a new ListMembersUseCase.
func NewListMembersUseCase(repo domain.MembershipRepository) *ListMembersUseCase {
	return &ListMembersUseCase{membershipRepo: repo}
}

// ByTenant returns all memberships for a given tenantID.
func (uc *ListMembersUseCase) ByTenant(ctx context.Context, tenantID string) ([]*domain.Membership, error) {
	memberships, err := uc.membershipRepo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return memberships, nil
}

// ByUser returns all memberships a user holds across tenants.
func (uc *ListMembersUseCase) ByUser(ctx context.Context, userID string) ([]*domain.Membership, error) {
	memberships, err := uc.membershipRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return memberships, nil
}
