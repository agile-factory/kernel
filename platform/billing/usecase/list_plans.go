// Package usecase contains billing application use cases.
package usecase

import (
	"context"

	"github.com/agile-factory/kernel/platform/billing/domain"
)

// ListPlansOutput wraps the list of plans.
type ListPlansOutput struct {
	Plans []*domain.Plan
}

// ListPlansUseCase returns all available SaaS plans.
type ListPlansUseCase struct {
	repo domain.PlanRepository
}

// NewListPlansUseCase builds the use case.
func NewListPlansUseCase(repo domain.PlanRepository) *ListPlansUseCase {
	return &ListPlansUseCase{repo: repo}
}

// Execute returns all plans ordered by the repository.
func (uc *ListPlansUseCase) Execute(ctx context.Context) (ListPlansOutput, error) {
	plans, err := uc.repo.FindAll(ctx)
	if err != nil {
		return ListPlansOutput{}, err
	}
	return ListPlansOutput{Plans: plans}, nil
}
