package usecase_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/agile-factory/kernel/kernel/events"
	"github.com/agile-factory/kernel/platform/audit/domain"
	"github.com/agile-factory/kernel/platform/audit/usecase"
)

// ─── Mock AuditRepository ─────────────────────────────────────────────────────

type mockAuditRepo struct {
	logs    []*domain.AuditLog
	saveErr error
	listErr error
}

func (m *mockAuditRepo) Save(_ context.Context, log *domain.AuditLog) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.logs = append(m.logs, log)
	return nil
}

func (m *mockAuditRepo) ListByTenant(_ context.Context, tenantID string, limit, offset int) ([]*domain.AuditLog, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var result []*domain.AuditLog
	for _, l := range m.logs {
		if l.TenantID == tenantID {
			result = append(result, l)
		}
	}
	if offset >= len(result) {
		return []*domain.AuditLog{}, nil
	}
	result = result[offset:]
	if limit < len(result) {
		result = result[:limit]
	}
	return result, nil
}

func (m *mockAuditRepo) ListAll(_ context.Context, limit, offset int) ([]*domain.AuditLog, error) {
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

func (m *mockAuditRepo) ListByTenantCursor(_ context.Context, tenantID string, limit int, after *domain.CursorToken) ([]*domain.AuditLog, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	var result []*domain.AuditLog
	for _, l := range m.logs {
		if l.TenantID != tenantID {
			continue
		}
		if after != nil {
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

func (m *mockAuditRepo) ListAllCursor(_ context.Context, limit int, after *domain.CursorToken) ([]*domain.AuditLog, error) {
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

// ─── ConsumeAuditLogUseCase ────────────────────────────────────────────────────

func TestConsumeAuditLog_PersistsLog(t *testing.T) {
	repo := &mockAuditRepo{}
	uc := usecase.NewConsumeAuditLogUseCase(repo)

	ev := events.Event{
		ID:   "outbox-uuid-001",
		Type: "audit.log_created",
		Payload: map[string]any{
			"tenant_id":   "tenant-001",
			"actor_id":    "user-001",
			"action":      "USER_LOGGED_IN",
			"target_type": "user",
			"target_id":   "user-001",
			"metadata":    map[string]any{"ip": "127.0.0.1"},
			"occurred_at": time.Now().UTC().Format(time.RFC3339Nano),
		},
		Meta: events.EventMeta{TenantID: "tenant-001"},
	}

	if err := uc.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle() unexpected error: %v", err)
	}
	if len(repo.logs) != 1 {
		t.Fatalf("expected 1 log saved, got %d", len(repo.logs))
	}
	got := repo.logs[0]
	if got.TenantID != "tenant-001" {
		t.Errorf("TenantID: want tenant-001, got %s", got.TenantID)
	}
	if got.Action != domain.ACTION_USER_LOGGED_IN {
		t.Errorf("Action: want USER_LOGGED_IN, got %s", got.Action)
	}
	if got.OutboxEventID != "outbox-uuid-001" {
		t.Errorf("OutboxEventID: want outbox-uuid-001, got %s", got.OutboxEventID)
	}
}

func TestConsumeAuditLog_IdempotentOnSaveError(t *testing.T) {
	// When repo.Save returns an error, Handle surfaces it so the dispatcher
	// can retry. The duplicate will be silently discarded on a successful
	// retry via ON CONFLICT (outbox_event_id) DO NOTHING in SQL.
	repo := &mockAuditRepo{saveErr: fmt.Errorf("db: connection refused")}
	uc := usecase.NewConsumeAuditLogUseCase(repo)

	ev := events.Event{
		ID:   "outbox-uuid-002",
		Type: "audit.log_created",
		Payload: map[string]any{
			"tenant_id":   "tenant-001",
			"actor_id":    "user-001",
			"action":      "USER_REGISTERED",
			"occurred_at": time.Now().UTC().Format(time.RFC3339Nano),
		},
	}

	if err := uc.Handle(context.Background(), ev); err == nil {
		t.Error("Handle() expected error on repo failure, got nil")
	}
}

func TestConsumeAuditLog_EmptyMetadataDefaultsToMap(t *testing.T) {
	repo := &mockAuditRepo{}
	uc := usecase.NewConsumeAuditLogUseCase(repo)

	ev := events.Event{
		ID:   "outbox-uuid-003",
		Type: "audit.log_created",
		Payload: map[string]any{
			"tenant_id":   "t-1",
			"actor_id":    "u-1",
			"action":      "TENANT_CREATED",
			"occurred_at": time.Now().UTC().Format(time.RFC3339Nano),
			// metadata field absent
		},
	}
	if err := uc.Handle(context.Background(), ev); err != nil {
		t.Fatalf("Handle() unexpected error: %v", err)
	}
	if repo.logs[0].Metadata == nil {
		t.Error("Metadata should default to empty map, got nil")
	}
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
	repo := populatedRepo(10, "tenant-001")
	uc := usecase.NewListAuditLogsUseCase(repo)

	logs, err := uc.ByTenant(context.Background(), "tenant-001", 0, 0)
	if err != nil {
		t.Fatalf("ByTenant() error: %v", err)
	}
	if len(logs) != 10 {
		t.Errorf("expected 10 logs (all), got %d", len(logs))
	}
}

func TestByTenant_MaxLimitCapped(t *testing.T) {
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
