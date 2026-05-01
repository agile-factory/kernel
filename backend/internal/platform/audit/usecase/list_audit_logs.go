package usecase

import (
	"context"

	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/audit/domain"
)

// ListAuditLogsUseCase handles the retrieval of audit logs for a tenant.
type ListAuditLogsUseCase struct {
	repo domain.AuditRepository
}

// NewListAuditLogsUseCase creates a new ListAuditLogsUseCase.
func NewListAuditLogsUseCase(repo domain.AuditRepository) *ListAuditLogsUseCase {
	return &ListAuditLogsUseCase{repo: repo}
}

// ByTenant returns paginated audit logs for a given tenant.
// limit defaults to 50 if 0 is passed; max is capped at 200.
func (uc *ListAuditLogsUseCase) ByTenant(ctx context.Context, tenantID string, limit, offset int) ([]*domain.AuditLog, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	return uc.repo.ListByTenant(ctx, tenantID, limit, offset)
}

// All returns paginated audit logs across all tenants.
// Intended for platform super-admins only.
func (uc *ListAuditLogsUseCase) All(ctx context.Context, limit, offset int) ([]*domain.AuditLog, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	return uc.repo.ListAll(ctx, limit, offset)
}
