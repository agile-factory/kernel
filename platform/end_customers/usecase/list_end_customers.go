package usecase

import (
	"context"

	kernelauth "github.com/agile-factory/kernel/kernel/auth"
	"github.com/agile-factory/kernel/kernel/permission"
	"github.com/agile-factory/kernel/platform/end_customers/domain"
)

// ListEndCustomersInput carries the filters and pagination controls.
type ListEndCustomersInput struct {
	Cursor string
	Limit  int
	Status *domain.EndCustomerStatus
	Tag    string
	Search string
}

// ListEndCustomersOutput holds the page results and next cursor.
type ListEndCustomersOutput struct {
	Items      []*domain.EndCustomer
	NextCursor string
}

// ListEndCustomersUseCase returns a paginated list of end-customers.
type ListEndCustomersUseCase struct {
	repo    domain.EndCustomerRepository
	checker permission.Checker
}

// NewListEndCustomersUseCase builds the use case.
func NewListEndCustomersUseCase(repo domain.EndCustomerRepository, checker permission.Checker) *ListEndCustomersUseCase {
	return &ListEndCustomersUseCase{repo: repo, checker: checker}
}

// Execute returns one page of end-customers for the authenticated tenant.
func (uc *ListEndCustomersUseCase) Execute(ctx context.Context, input ListEndCustomersInput) (ListEndCustomersOutput, error) {
	if err := uc.checker.Require(ctx, permission.EndCustomerList); err != nil {
		return ListEndCustomersOutput{}, err
	}

	tenantID := kernelauth.TenantIDFromContext(ctx)

	limit := input.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	items, nextCursor, err := uc.repo.List(ctx, domain.ListParams{
		TenantID: tenantID,
		Cursor:   input.Cursor,
		Limit:    limit,
		Status:   input.Status,
		Tag:      input.Tag,
		Search:   input.Search,
	})
	if err != nil {
		return ListEndCustomersOutput{}, err
	}
	return ListEndCustomersOutput{Items: items, NextCursor: nextCursor}, nil
}
