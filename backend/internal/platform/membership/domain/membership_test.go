package domain_test

import (
	"testing"
	"time"

	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/membership/domain"
)

// ─── NewSelfMembership ────────────────────────────────────────────────────────

func TestNewSelfMembership_IsActive(t *testing.T) {
	m := domain.NewSelfMembership("mem-001", "user-001", "tenant-001", domain.MEMBER_ROLE_OWNER)

	if m.Status != domain.MEMBER_STATUS_ACTIVE {
		t.Errorf("expected ACTIVE, got %s", m.Status)
	}
}

func TestNewSelfMembership_JoinedAtIsSet(t *testing.T) {
	before := time.Now()
	m := domain.NewSelfMembership("mem-001", "user-001", "tenant-001", domain.MEMBER_ROLE_OWNER)
	after := time.Now()

	if m.JoinedAt == nil {
		t.Fatal("JoinedAt should not be nil for a self-membership")
	}
	if m.JoinedAt.Before(before) || m.JoinedAt.After(after) {
		t.Errorf("JoinedAt out of expected range: %v", m.JoinedAt)
	}
}

func TestNewSelfMembership_NoInviter(t *testing.T) {
	m := domain.NewSelfMembership("mem-001", "user-001", "tenant-001", domain.MEMBER_ROLE_VIEWER)
	if m.InvitedBy != "" {
		t.Errorf("InvitedBy should be empty for self-membership, got %q", m.InvitedBy)
	}
}

func TestNewSelfMembership_FieldsAreSet(t *testing.T) {
	m := domain.NewSelfMembership("mem-001", "user-001", "tenant-001", domain.MEMBER_ROLE_ADMIN)

	if m.ID != "mem-001" {
		t.Errorf("ID: got %q, want %q", m.ID, "mem-001")
	}
	if m.UserID != "user-001" {
		t.Errorf("UserID: got %q, want %q", m.UserID, "user-001")
	}
	if m.TenantID != "tenant-001" {
		t.Errorf("TenantID: got %q, want %q", m.TenantID, "tenant-001")
	}
	if m.Role != domain.MEMBER_ROLE_ADMIN {
		t.Errorf("Role: got %q, want ROLE_ADMIN", m.Role)
	}
}

// ─── NewInvitedMembership ─────────────────────────────────────────────────────

func TestNewInvitedMembership_IsPending(t *testing.T) {
	m := domain.NewInvitedMembership("mem-002", "user-002", "tenant-001", "admin-001", domain.MEMBER_ROLE_AGENT)

	if m.Status != domain.MEMBER_STATUS_PENDING {
		t.Errorf("expected PENDING, got %s", m.Status)
	}
}

func TestNewInvitedMembership_JoinedAtIsNil(t *testing.T) {
	m := domain.NewInvitedMembership("mem-002", "user-002", "tenant-001", "admin-001", domain.MEMBER_ROLE_AGENT)
	if m.JoinedAt != nil {
		t.Error("JoinedAt should be nil for an invited (pending) membership")
	}
}

func TestNewInvitedMembership_InviterSet(t *testing.T) {
	m := domain.NewInvitedMembership("mem-002", "user-002", "tenant-001", "admin-001", domain.MEMBER_ROLE_AGENT)
	if m.InvitedBy != "admin-001" {
		t.Errorf("InvitedBy: got %q, want %q", m.InvitedBy, "admin-001")
	}
}

// ─── Accept ───────────────────────────────────────────────────────────────────

func TestAccept_TransitionsPendingToActive(t *testing.T) {
	m := domain.NewInvitedMembership("mem-003", "user-003", "tenant-001", "admin-001", domain.MEMBER_ROLE_AGENT)

	if err := m.Accept(); err != nil {
		t.Fatalf("Accept() unexpected error: %v", err)
	}
	if m.Status != domain.MEMBER_STATUS_ACTIVE {
		t.Errorf("after Accept: expected ACTIVE, got %s", m.Status)
	}
}

func TestAccept_SetsJoinedAt(t *testing.T) {
	m := domain.NewInvitedMembership("mem-003", "user-003", "tenant-001", "admin-001", domain.MEMBER_ROLE_AGENT)
	before := time.Now()
	_ = m.Accept()
	after := time.Now()

	if m.JoinedAt == nil {
		t.Fatal("JoinedAt should be set after Accept()")
	}
	if m.JoinedAt.Before(before) || m.JoinedAt.After(after) {
		t.Errorf("JoinedAt out of expected range: %v", m.JoinedAt)
	}
}

func TestAccept_FailsIfAlreadyActive(t *testing.T) {
	// SelfMembership démarre en ACTIVE
	m := domain.NewSelfMembership("mem-004", "user-004", "tenant-001", domain.MEMBER_ROLE_VIEWER)

	err := m.Accept()
	if err == nil {
		t.Fatal("Accept() on ACTIVE membership should return an error")
	}
	if err != domain.ErrMembershipNotPending {
		t.Errorf("expected ErrMembershipNotPending, got %v", err)
	}
}

func TestAccept_FailsIfAlreadyRevoked(t *testing.T) {
	m := domain.NewInvitedMembership("mem-005", "user-005", "tenant-001", "admin-001", domain.MEMBER_ROLE_VIEWER)
	_ = m.Accept()
	_ = m.Revoke()

	err := m.Accept()
	if err == nil {
		t.Fatal("Accept() on REVOKED membership should return an error")
	}
}

// ─── Revoke ───────────────────────────────────────────────────────────────────

func TestRevoke_ActiveMembership(t *testing.T) {
	m := domain.NewSelfMembership("mem-006", "user-006", "tenant-001", domain.MEMBER_ROLE_AGENT)

	if err := m.Revoke(); err != nil {
		t.Fatalf("Revoke() unexpected error: %v", err)
	}
	if m.Status != domain.MEMBER_STATUS_REVOKED {
		t.Errorf("after Revoke: expected REVOKED, got %s", m.Status)
	}
}

func TestRevoke_AlreadyRevoked(t *testing.T) {
	m := domain.NewSelfMembership("mem-007", "user-007", "tenant-001", domain.MEMBER_ROLE_AGENT)
	_ = m.Revoke()

	err := m.Revoke()
	if err == nil {
		t.Fatal("Revoke() on already-revoked membership should return an error")
	}
	if err != domain.ErrMembershipAlreadyRevoked {
		t.Errorf("expected ErrMembershipAlreadyRevoked, got %v", err)
	}
}

func TestRevoke_PendingMembership(t *testing.T) {
	m := domain.NewInvitedMembership("mem-008", "user-008", "tenant-001", "admin-001", domain.MEMBER_ROLE_VIEWER)

	// Révoquer un membership PENDING doit aussi être autorisé (il n'est pas encore ACTIVE)
	if err := m.Revoke(); err != nil {
		t.Fatalf("Revoke() on PENDING should succeed, got: %v", err)
	}
}

// ─── Predicate helpers ────────────────────────────────────────────────────────

func TestIsActive(t *testing.T) {
	active := domain.NewSelfMembership("m", "u", "t", domain.MEMBER_ROLE_VIEWER)
	pending := domain.NewInvitedMembership("m2", "u2", "t", "admin", domain.MEMBER_ROLE_AGENT)
	revoked := domain.NewSelfMembership("m3", "u3", "t", domain.MEMBER_ROLE_VIEWER)
	_ = revoked.Revoke()

	if !active.IsActive() {
		t.Error("ACTIVE membership should return true for IsActive()")
	}
	if pending.IsActive() {
		t.Error("PENDING membership should return false for IsActive()")
	}
	if revoked.IsActive() {
		t.Error("REVOKED membership should return false for IsActive()")
	}
}

func TestIsAdmin(t *testing.T) {
	owner := domain.NewSelfMembership("m", "u", "t", domain.MEMBER_ROLE_OWNER)
	admin := domain.NewSelfMembership("m2", "u2", "t", domain.MEMBER_ROLE_ADMIN)
	agent := domain.NewSelfMembership("m3", "u3", "t", domain.MEMBER_ROLE_AGENT)
	viewer := domain.NewSelfMembership("m4", "u4", "t", domain.MEMBER_ROLE_VIEWER)

	if !owner.IsAdmin() {
		t.Error("ROLE_OWNER should be considered admin")
	}
	if !admin.IsAdmin() {
		t.Error("ROLE_ADMIN should be considered admin")
	}
	if agent.IsAdmin() {
		t.Error("ROLE_AGENT should NOT be admin")
	}
	if viewer.IsAdmin() {
		t.Error("ROLE_VIEWER should NOT be admin")
	}
}

func TestIsOwner(t *testing.T) {
	owner := domain.NewSelfMembership("m", "u", "t", domain.MEMBER_ROLE_OWNER)
	admin := domain.NewSelfMembership("m2", "u2", "t", domain.MEMBER_ROLE_ADMIN)

	if !owner.IsOwner() {
		t.Error("ROLE_OWNER should return true for IsOwner()")
	}
	if admin.IsOwner() {
		t.Error("ROLE_ADMIN should return false for IsOwner()")
	}
}
