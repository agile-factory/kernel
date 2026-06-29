package usecase

import (
	"context"

	kernelauth "github.com/agile-factory/kernel/kernel/auth"
	"github.com/agile-factory/kernel/kernel/permission"
	"github.com/agile-factory/kernel/platform/end_customers/domain"
)

// GetEndCustomerOutput wraps the fetched end-customer.
type GetEndCustomerOutput struct {
	EndCustomer *domain.EndCustomer
}

// GetEndCustomerUseCase fetches a single end-customer by ID.
type GetEndCustomerUseCase struct {
	repo    domain.EndCustomerRepository
	checker permission.Checker
}

// NewGetEndCustomerUseCase builds the use case.
func NewGetEndCustomerUseCase(repo domain.EndCustomerRepository, checker permission.Checker) *GetEndCustomerUseCase {
	return &GetEndCustomerUseCase{repo: repo, checker: checker}
}

// Execute fetches the end-customer identified by id for the authenticated tenant.
func (uc *GetEndCustomerUseCase) Execute(ctx context.Context, id string) (GetEndCustomerOutput, error) {
	if err := uc.checker.Require(ctx, permission.EndCustomerList); err != nil {
		return GetEndCustomerOutput{}, err
	}

	tenantID := kernelauth.TenantIDFromContext(ctx)
	ec, err := uc.repo.FindByID(ctx, tenantID, id)
	if err != nil {
		return GetEndCustomerOutput{}, err
	}
	return GetEndCustomerOutput{EndCustomer: ec}, nil
}
