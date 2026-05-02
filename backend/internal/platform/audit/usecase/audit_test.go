package usecase_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/audit/domain"
	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/audit/usecase"
)

// ─── Mock AuditRepository ─────────────────────────────────────────────────────

type mockAuditRepo struct {
	logs     []*domain.AuditLog
	saveErr  error
	listErr  error
}

func (m *mockAuditRepo) Save(ctx context.Context, log *domain.AuditLog) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.logs = append(m.logs, log)
	return nil
}

func (m *mockAuditRepo) ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]*domain.AuditLog, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var result []*domain.AuditLog
	for _, l := range m.logs {
		if l.TenantID == tenantID {
			result = append(result, l)
		}
	}
	// Apply offset + limit
	if offset >= len(result) {
		return []*domain.AuditLog{}, nil
	}
	result = result[offset:]
	if limit < len(result) {
		result = result[:limit]
	}
	return result, nil
}

func (m *mockAuditRepo) ListAll(ctx context.Context, limit, offset int) ([]*domain.AuditLog, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	result := make([]*domain.AuditLog, len(m.logs))
	copy(result, m.logs)
	if offset >= len(result) {
		return []*domain.AuditLog{}, nil
	}
	result = result[offset:]
	if limit < len(result) {
		result = result[:limit]
	}
	return result, nil
}

func (m *mockAuditRepo) ListByTenantCursor(ctx context.Context, tenantID string, limit int, after *domain.CursorToken) ([]*domain.AuditLog, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var result []*domain.AuditLog
	for _, l := range m.logs {
		if l.TenantID != tenantID {
			continue
		}
		if after != nil {
			// Keyset: occurred_at < after.OccurredAt OR (occurred_at == after.OccurredAt AND id > after.ID)
			if l.OccurredAt.After(after.OccurredAt) {
				continue
			}
			if l.OccurredAt.Equal(after.OccurredAt) && l.ID <= after.ID {
				continue
			}
		}
		result = append(result, l)
	}
	if limit < len(result) {
		result = result[:limit]
	}
	return result, nil
}

func (m *mockAuditRepo) ListAllCursor(ctx context.Context, limit int, after *domain.CursorToken) ([]*domain.AuditLog, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var result []*domain.AuditLog
	for _, l := range m.logs {
		if after != nil {
			if l.OccurredAt.After(after.OccurredAt) {
				continue
			}
		}
		result = append(result, l)
	}
	if limit < len(result) {
		result = result[:limit]
	}
	return result, nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func makeLog(id, tenantID string, t time.Time) *domain.AuditLog {
	return &domain.AuditLog{
		ID:         id,
		TenantID:   tenantID,
		Action:     domain.ACTION_USER_REGISTERED,
		ActorID:    "actor-001",
		TargetType: "user",
		TargetID:   id,
		Metadata:   map[string]any{},
		OccurredAt: t,
	}
}

func populatedRepo(n int, tenantID string) *mockAuditRepo {
	repo := &mockAuditRepo{}
	base := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < n; i++ {
		repo.logs = append(repo.logs, makeLog(
			fmt.Sprintf("log-%03d", i),
			tenantID,
			base.Add(time.Duration(i)*time.Minute),
		))
	}
	return repo
}

// ─── LogEventUseCase ─────────────────────────────────────────────────────────

func TestLogEvent_SavesSuccessfully(t *testing.T) {
	repo := &mockAuditRepo{}
	uc := usecase.NewLogEventUseCase(repo)

	uc.Execute(context.Background(), usecase.LogEventParams{
		TenantID:   "tenant-001",
		ActorID:    "user-001",
		Action:     domain.ACTION_USER_REGISTERED,
		TargetType: "user",
		TargetID:   "user-001",
		Metadata:   map[string]any{"email": "alice@agile.io"},
	})

	if len(repo.logs) != 1 {
		t.Errorf("expected 1 log saved, got %d", len(repo.logs))
	}
}

func TestLogEvent_FireAndForget_NoError(t *testing.T) {
	// Même si le repo échoue, LogEvent ne doit pas paniquer ni retourner d'erreur.
	repo := &mockAuditRepo{saveErr: fmt.Errorf("db: connection refused")}
	uc := usecase.NewLogEventUseCase(repo)

	// Ne doit pas paniquer
	uc.Execute(context.Background(), usecase.LogEventParams{
		TenantID: "tenant-001",
		ActorID:  "user-001",
		Action:   domain.ACTION_USER_LOGGED_IN,
	})
}

// ─── ListAuditLogsUseCase — v1 Offset ─────────────────────────────────────────

func TestByTenant_ReturnsCorrectLogs(t *testing.T) {
	repo := populatedRepo(10, "tenant-001")
	uc := usecase.NewListAuditLogsUseCase(repo)

	logs, err := uc.ByTenant(context.Background(), "tenant-001", 5, 0)
	if err != nil {
		t.Fatalf("ByTenant() error: %v", err)
	}
	if len(logs) != 5 {
		t.Errorf("expected 5 logs, got %d", len(logs))
	}
}

func TestByTenant_DefaultLimit(t *testing.T) {
	// Limit = 0 doit être ramenée à 50.
	repo := populatedRepo(10, "tenant-001")
	uc := usecase.NewListAuditLogsUseCase(repo)

	logs, err := uc.ByTenant(context.Background(), "tenant-001", 0, 0)
	if err != nil {
		t.Fatalf("ByTenant() error: %v", err)
	}
	// Le repo mock retourne tous les logs du tenant (10), limit 50 les garde tous.
	if len(logs) != 10 {
		t.Errorf("expected 10 logs (all), got %d", len(logs))
	}
}

func TestByTenant_MaxLimitCapped(t *testing.T) {
	// Limit > 200 doit être ramenée à 200.
	repo := populatedRepo(10, "tenant-001")
	uc := usecase.NewListAuditLogsUseCase(repo)

	logs, err := uc.ByTenant(context.Background(), "tenant-001", 9999, 0)
	if err != nil {
		t.Fatalf("ByTenant() error: %v", err)
	}
	if len(logs) != 10 {
		t.Errorf("expected 10 logs (capped), got %d", len(logs))
	}
}

func TestByTenant_OffsetPagination(t *testing.T) {
	repo := populatedRepo(10, "tenant-001")
	uc := usecase.NewListAuditLogsUseCase(repo)

	logs, err := uc.ByTenant(context.Background(), "tenant-001", 5, 5)
	if err != nil {
		t.Fatalf("ByTenant() error: %v", err)
	}
	if len(logs) != 5 {
		t.Errorf("expected 5 logs (page 2), got %d", len(logs))
	}
}

func TestByTenant_DifferentTenantsIsolated(t *testing.T) {
	repo := populatedRepo(5, "tenant-001")
	// Ajouter des logs pour tenant-002
	for i := 0; i < 3; i++ {
		repo.logs = append(repo.logs, makeLog(
			fmt.Sprintf("other-%03d", i), "tenant-002",
			time.Now(),
		))
	}
	uc := usecase.NewListAuditLogsUseCase(repo)

	logs, err := uc.ByTenant(context.Background(), "tenant-001", 50, 0)
	if err != nil {
		t.Fatalf("ByTenant() error: %v", err)
	}
	for _, l := range logs {
		if l.TenantID != "tenant-001" {
			t.Errorf("Got log from wrong tenant: %s", l.TenantID)
		}
	}
}

func TestAll_ReturnsCrossTenantsLogs(t *testing.T) {
	repo := &mockAuditRepo{}
	repo.logs = append(repo.logs, makeLog("l1", "tenant-001", time.Now()))
	repo.logs = append(repo.logs, makeLog("l2", "tenant-002", time.Now()))
	uc := usecase.NewListAuditLogsUseCase(repo)

	logs, err := uc.All(context.Background(), 50, 0)
	if err != nil {
		t.Fatalf("All() error: %v", err)
	}
	if len(logs) != 2 {
		t.Errorf("expected 2 logs from all tenants, got %d", len(logs))
	}
}

// ─── ListAuditLogsUseCase — v2 Cursor ─────────────────────────────────────────

func TestByTenantCursor_FirstPage(t *testing.T) {
	repo := populatedRepo(10, "tenant-001")
	uc := usecase.NewListAuditLogsUseCase(repo)

	page, err := uc.ByTenantCursor(context.Background(), "tenant-001", 5, "")
	if err != nil {
		t.Fatalf("ByTenantCursor() error: %v", err)
	}
	if len(page.Logs) != 5 {
		t.Errorf("expected 5 logs, got %d", len(page.Logs))
	}
}

func TestByTenantCursor_HasMoreWhenMorePagesExist(t *testing.T) {
	repo := populatedRepo(10, "tenant-001")
	uc := usecase.NewListAuditLogsUseCase(repo)

	page, err := uc.ByTenantCursor(context.Background(), "tenant-001", 5, "")
	if err != nil {
		t.Fatalf("ByTenantCursor() error: %v", err)
	}
	if !page.HasMore {
		t.Error("HasMore should be true when there are more pages")
	}
}

func TestByTenantCursor_LastPageHasNoMore(t *testing.T) {
	repo := populatedRepo(3, "tenant-001")
	uc := usecase.NewListAuditLogsUseCase(repo)

	page, err := uc.ByTenantCursor(context.Background(), "tenant-001", 10, "")
	if err != nil {
		t.Fatalf("ByTenantCursor() error: %v", err)
	}
	if page.HasMore {
		t.Error("HasMore should be false on the last page")
	}
	if page.NextCursor != "" {
		t.Errorf("NextCursor should be empty on the last page, got %q", page.NextCursor)
	}
}

func TestByTenantCursor_InvalidCursorReturnsError(t *testing.T) {
	repo := populatedRepo(5, "tenant-001")
	uc := usecase.NewListAuditLogsUseCase(repo)

	_, err := uc.ByTenantCursor(context.Background(), "tenant-001", 5, "!!!invalid-cursor!!!")
	if err == nil {
		t.Error("ByTenantCursor() should return error for an invalid cursor")
	}
}

func TestAllCursor_ReturnsAllTenants(t *testing.T) {
	repo := &mockAuditRepo{}
	repo.logs = append(repo.logs, makeLog("l1", "tenant-001", time.Now()))
	repo.logs = append(repo.logs, makeLog("l2", "tenant-002", time.Now()))
	uc := usecase.NewListAuditLogsUseCase(repo)

	page, err := uc.AllCursor(context.Background(), 50, "")
	if err != nil {
		t.Fatalf("AllCursor() error: %v", err)
	}
	if len(page.Logs) != 2 {
		t.Errorf("expected 2 logs, got %d", len(page.Logs))
	}
}

func TestAllCursor_InvalidCursorReturnsError(t *testing.T) {
	repo := populatedRepo(3, "tenant-001")
	uc := usecase.NewListAuditLogsUseCase(repo)

	_, err := uc.AllCursor(context.Background(), 10, "not-valid-base64!!!")
	if err == nil {
		t.Error("AllCursor() should return error for an invalid cursor")
	}
}
