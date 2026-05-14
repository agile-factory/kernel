package usecase

import (
	"context"
	"fmt"

	"github.com/AgileFactory/kernel/platform/audit/domain"
)

// ListAuditLogsUseCase handles retrieval of audit logs.
// It exposes both the v1 offset-based API and the v2 cursor-based API.
type ListAuditLogsUseCase struct {
	repo domain.AuditRepository
}

// NewListAuditLogsUseCase creates a new ListAuditLogsUseCase.
func NewListAuditLogsUseCase(repo domain.AuditRepository) *ListAuditLogsUseCase {
	return &ListAuditLogsUseCase{repo: repo}
}

// ── v1 — offset-based ─────────────────────────────────────────────────────────

// ByTenant returns paginated audit logs for a given tenant.
// limit defaults to 50 if 0 is passed; max is capped at 200.
func (uc *ListAuditLogsUseCase) ByTenant(ctx context.Context, tenantID string, limit, offset int) ([]*domain.AuditLog, error) {
	limit = clampLimit(limit)
	return uc.repo.ListByTenant(ctx, tenantID, limit, offset)
}

// All returns paginated audit logs across all tenants.
// Intended for platform super-admins only.
func (uc *ListAuditLogsUseCase) All(ctx context.Context, limit, offset int) ([]*domain.AuditLog, error) {
	limit = clampLimit(limit)
	return uc.repo.ListAll(ctx, limit, offset)
}

// ── v2 — cursor-based ─────────────────────────────────────────────────────────

// CursorPage is the result returned by the cursor-based use case methods.
type CursorPage struct {
	Logs       []*domain.AuditLog
	NextCursor string // Base64url-encoded; empty string on the last page
	HasMore    bool
}

// ByTenantCursor returns a cursor-based page of audit logs for a specific tenant.
//
// afterCursor is the opaque next_cursor value from the previous response.
// Pass an empty string to request the first page.
//
// limit defaults to 50 and is capped at 200.
func (uc *ListAuditLogsUseCase) ByTenantCursor(
	ctx context.Context, tenantID string, limit int, afterCursor string,
) (*CursorPage, error) {
	limit = clampLimit(limit)

	after, err := parseCursor(afterCursor)
	if err != nil {
		return nil, err
	}

	// Fetch one extra row to determine whether a next page exists.
	rows, err := uc.repo.ListByTenantCursor(ctx, tenantID, limit+1, after)
	if err != nil {
		return nil, fmt.Errorf("list by tenant cursor: %w", err)
	}

	return buildCursorPage(rows, limit)
}

// AllCursor returns a cursor-based page of audit logs across all tenants.
//
// afterCursor is the opaque next_cursor value from the previous response.
// Pass an empty string to request the first page.
//
// limit defaults to 50 and is capped at 200.
func (uc *ListAuditLogsUseCase) AllCursor(
	ctx context.Context, limit int, afterCursor string,
) (*CursorPage, error) {
	limit = clampLimit(limit)

	after, err := parseCursor(afterCursor)
	if err != nil {
		return nil, err
	}

	rows, err := uc.repo.ListAllCursor(ctx, limit+1, after)
	if err != nil {
		return nil, fmt.Errorf("list all cursor: %w", err)
	}

	return buildCursorPage(rows, limit)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func clampLimit(n int) int {
	if n <= 0 {
		return 50
	}
	if n > 200 {
		return 200
	}
	return n
}

// parseCursor decodes the afterCursor string into a *domain.CursorToken.
// Returns nil (no error) when afterCursor is empty — this signals "first page".
// All decode failures are wrapped with domain.ErrInvalidCursor so callers can
// distinguish a bad client input (HTTP 400) from a repo failure (HTTP 500).
func parseCursor(afterCursor string) (*domain.CursorToken, error) {
	if afterCursor == "" {
		return nil, nil
	}
	tok, err := domain.DecodeCursor(afterCursor)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidCursor, err)
	}
	return tok, nil
}

// buildCursorPage applies the limit+1 sentinel logic:
//   - If len(rows) > limit → more pages exist; trim the sentinel row
//   - Encode the last retained row as next_cursor
//
// Surfaces the (theoretically impossible) EncodeCursor error rather than
// silencing it: returning has_more=true alongside an empty next_cursor
// would leave the client unable to fetch the next page.
func buildCursorPage(rows []*domain.AuditLog, limit int) (*CursorPage, error) {
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit] // drop the sentinel row
	}

	var nextCursor string
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		tok := domain.CursorFromLog(last)
		s, err := domain.EncodeCursor(tok)
		if err != nil {
			return nil, fmt.Errorf("encode next cursor: %w", err)
		}
		nextCursor = s
	}

	return &CursorPage{
		Logs:       rows,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}
