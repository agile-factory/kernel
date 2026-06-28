package usecase

import (
	"context"

	apperrors "github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/platform/billing/domain"
)

// GetSubscriptionInput carries the lookup key.
type GetSubscriptionInput struct {
	TenantID string
}

// GetSubscriptionOutput wraps the subscription.
type GetSubscriptionOutput struct {
	Subscription *domain.Subscription
}

// GetSubscriptionUseCase retrieves the active subscription for a tenant.
type GetSubscriptionUseCase struct {
	repo domain.SubscriptionRepository
}

// NewGetSubscriptionUseCase builds the use case.
func NewGetSubscriptionUseCase(repo domain.SubscriptionRepository) *GetSubscriptionUseCase {
	return &GetSubscriptionUseCase{repo: repo}
}

// Execute returns the subscription or a 404 error if absent.
func (uc *GetSubscriptionUseCase) Execute(ctx context.Context, input GetSubscriptionInput) (GetSubscriptionOutput, error) {
	if input.TenantID == "" {
		return GetSubscriptionOutput{}, apperrors.BadRequest("tenantID is required")
	}
	sub, err := uc.repo.FindByTenantID(ctx, input.TenantID)
	if err != nil {
		return GetSubscriptionOutput{}, err
	}
	return GetSubscriptionOutput{Subscription: sub}, nil
}
