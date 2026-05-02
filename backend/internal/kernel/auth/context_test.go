package auth_test

import (
	"context"
	"testing"

	kernelauth "github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/kernel/auth"
)

// ─── UserID ───────────────────────────────────────────────────────────────────

func TestWithUserID_StoresValue(t *testing.T) {
	ctx := kernelauth.WithUserID(context.Background(), "user-001")
	got := kernelauth.UserIDFromContext(ctx)
	if got != "user-001" {
		t.Errorf("UserIDFromContext: got %q, want %q", got, "user-001")
	}
}

func TestUserIDFromContext_EmptyWhenNotSet(t *testing.T) {
	got := kernelauth.UserIDFromContext(context.Background())
	if got != "" {
		t.Errorf("UserIDFromContext on empty context: got %q, want empty string", got)
	}
}

func TestWithUserID_OverwritesPreviousValue(t *testing.T) {
	ctx := kernelauth.WithUserID(context.Background(), "user-001")
	ctx = kernelauth.WithUserID(ctx, "user-002")
	got := kernelauth.UserIDFromContext(ctx)
	if got != "user-002" {
		t.Errorf("UserIDFromContext after overwrite: got %q, want %q", got, "user-002")
	}
}

func TestWithUserID_DoesNotAffectParentContext(t *testing.T) {
	parent := context.Background()
	_ = kernelauth.WithUserID(parent, "user-001")
	got := kernelauth.UserIDFromContext(parent)
	if got != "" {
		t.Errorf("UserIDFromContext on parent: got %q, want empty string", got)
	}
}

// ─── TenantID ─────────────────────────────────────────────────────────────────

func TestWithTenantID_StoresValue(t *testing.T) {
	ctx := kernelauth.WithTenantID(context.Background(), "tenant-001")
	got := kernelauth.TenantIDFromContext(ctx)
	if got != "tenant-001" {
		t.Errorf("TenantIDFromContext: got %q, want %q", got, "tenant-001")
	}
}

func TestTenantIDFromContext_EmptyWhenNotSet(t *testing.T) {
	got := kernelauth.TenantIDFromContext(context.Background())
	if got != "" {
		t.Errorf("TenantIDFromContext on empty context: got %q, want empty string", got)
	}
}

func TestWithTenantID_OverwritesPreviousValue(t *testing.T) {
	ctx := kernelauth.WithTenantID(context.Background(), "tenant-001")
	ctx = kernelauth.WithTenantID(ctx, "tenant-002")
	got := kernelauth.TenantIDFromContext(ctx)
	if got != "tenant-002" {
		t.Errorf("TenantIDFromContext after overwrite: got %q, want %q", got, "tenant-002")
	}
}

// ─── PlatformRole ─────────────────────────────────────────────────────────────

func TestWithPlatformRole_StoresValue(t *testing.T) {
	ctx := kernelauth.WithPlatformRole(context.Background(), "ROLE_ADMIN")
	got := kernelauth.PlatformRoleFromContext(ctx)
	if got != "ROLE_ADMIN" {
		t.Errorf("PlatformRoleFromContext: got %q, want %q", got, "ROLE_ADMIN")
	}
}

func TestPlatformRoleFromContext_EmptyWhenNotSet(t *testing.T) {
	got := kernelauth.PlatformRoleFromContext(context.Background())
	if got != "" {
		t.Errorf("PlatformRoleFromContext on empty context: got %q, want empty string", got)
	}
}

func TestWithPlatformRole_OverwritesPreviousValue(t *testing.T) {
	ctx := kernelauth.WithPlatformRole(context.Background(), "ROLE_VIEWER")
	ctx = kernelauth.WithPlatformRole(ctx, "ROLE_ADMIN")
	got := kernelauth.PlatformRoleFromContext(ctx)
	if got != "ROLE_ADMIN" {
		t.Errorf("PlatformRoleFromContext after overwrite: got %q, want %q", got, "ROLE_ADMIN")
	}
}

// ─── IsPlatformAdmin ──────────────────────────────────────────────────────────

func TestIsPlatformAdmin_TrueForROLE_ADMIN(t *testing.T) {
	ctx := kernelauth.WithPlatformRole(context.Background(), "ROLE_ADMIN")
	if !kernelauth.IsPlatformAdmin(ctx) {
		t.Error("IsPlatformAdmin should be true for ROLE_ADMIN")
	}
}

func TestIsPlatformAdmin_FalseWhenNotSet(t *testing.T) {
	if kernelauth.IsPlatformAdmin(context.Background()) {
		t.Error("IsPlatformAdmin should be false when no role is set")
	}
}

func TestIsPlatformAdmin_FalseForOtherRoles(t *testing.T) {
	roles := []string{"ROLE_VIEWER", "ROLE_AGENT", "ROLE_OWNER", "ROLE_GUEST", ""}
	for _, role := range roles {
		t.Run(role, func(t *testing.T) {
			ctx := kernelauth.WithPlatformRole(context.Background(), role)
			if kernelauth.IsPlatformAdmin(ctx) {
				t.Errorf("IsPlatformAdmin should be false for role %q", role)
			}
		})
	}
}

// ─── Key isolation ────────────────────────────────────────────────────────────

// Verify that UserID, TenantID, and PlatformRole keys do not collide with
// each other — adding one must not affect the others.
func TestContextKeys_DoNotCollide(t *testing.T) {
	ctx := context.Background()
	ctx = kernelauth.WithUserID(ctx, "user-abc")
	ctx = kernelauth.WithTenantID(ctx, "tenant-xyz")
	ctx = kernelauth.WithPlatformRole(ctx, "ROLE_ADMIN")

	if uid := kernelauth.UserIDFromContext(ctx); uid != "user-abc" {
		t.Errorf("UserID: got %q, want %q", uid, "user-abc")
	}
	if tid := kernelauth.TenantIDFromContext(ctx); tid != "tenant-xyz" {
		t.Errorf("TenantID: got %q, want %q", tid, "tenant-xyz")
	}
	if role := kernelauth.PlatformRoleFromContext(ctx); role != "ROLE_ADMIN" {
		t.Errorf("PlatformRole: got %q, want %q", role, "ROLE_ADMIN")
	}
}
