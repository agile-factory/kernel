package permission_test

import (
	"context"
	"errors"
	"testing"
	"time"

	kernelauth "github.com/agile-factory/kernel/kernel/auth"
	apperrors "github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/kernel/permission"
	"github.com/agile-factory/kernel/platform/membership/domain"
)

// stubMemberRepo is an in-memory MembershipRepository for tests.
type stubMemberRepo struct {
	members map[string]*domain.Membership // key: userID+":"+tenantID
	err     error
}

func (s *stubMemberRepo) FindByUserAndTenant(_ context.Context, userID, tenantID string) (*domain.Membership, error) {
	if s.err != nil {
		return nil, s.err
	}
	m, ok := s.members[userID+":"+tenantID]
	if !ok {
		return nil, apperrors.NotFound("membership not found")
	}
	return m, nil
}

// Implement remaining interface methods as no-ops (not exercised here).
func (s *stubMemberRepo) Save(_ context.Context, _ *domain.Membership) error { return nil }
func (s *stubMemberRepo) FindByID(_ context.Context, _ string) (*domain.Membership, error) {
	return nil, nil
}
func (s *stubMemberRepo) ListByTenant(_ context.Context, _ string) ([]*domain.Membership, error) {
	return nil, nil
}
func (s *stubMemberRepo) ListByUser(_ context.Context, _ string) ([]*domain.Membership, error) {
	return nil, nil
}
func (s *stubMemberRepo) UpdateStatus(_ context.Context, _ string, _ domain.MemberStatus, _ *time.Time) error {
	return nil
}

func activeMember(userID, tenantID string, role domain.MemberRole) *domain.Membership {
	now := time.Now()
	return &domain.Membership{
		ID:       "m-" + userID,
		UserID:   userID,
		TenantID: tenantID,
		Role:     role,
		Status:   domain.MEMBER_STATUS_ACTIVE,
		JoinedAt: &now,
	}
}

func ctxWith(userID, tenantID string) context.Context {
	ctx := context.Background()
	ctx = kernelauth.WithUserID(ctx, userID)
	ctx = kernelauth.WithTenantID(ctx, tenantID)
	return ctx
}

func ctxPlatformAdmin() context.Context {
	ctx := ctxWith("admin-user", "any-tenant")
	return kernelauth.WithPlatformRole(ctx, "ROLE_ADMIN")
}

func TestRequire_OwnerHasAllPermissions(t *testing.T) {
	repo := &stubMemberRepo{members: map[string]*domain.Membership{
		"u1:t1": activeMember("u1", "t1", domain.MEMBER_ROLE_OWNER),
	}}
	checker := permission.New(repo)
	ctx := ctxWith("u1", "t1")

	for _, perm := range []permission.Permission{
		permission.TenantView, permission.TenantUpdate, permission.TenantDelete,
		permission.MemberInvite, permission.MemberRevoke, permission.MemberList,
		permission.AuditView,
		permission.UserProfileView, permission.UserProfileUpdateOwn,
		permission.StorageUpload, permission.StorageDelete,
	} {
		if err := checker.Require(ctx, perm); err != nil {
			t.Errorf("OWNER should have %s, got error: %v", perm, err)
		}
	}
}

func TestRequire_AdminCannotDeleteTenant(t *testing.T) {
	repo := &stubMemberRepo{members: map[string]*domain.Membership{
		"u2:t1": activeMember("u2", "t1", domain.MEMBER_ROLE_ADMIN),
	}}
	checker := permission.New(repo)
	ctx := ctxWith("u2", "t1")

	if err := checker.Require(ctx, permission.TenantDelete); err == nil {
		t.Error("ADMIN should NOT have tenant:delete")
	}
	if err := checker.Require(ctx, permission.TenantUpdate); err != nil {
		t.Errorf("ADMIN should have tenant:update, got: %v", err)
	}
}

func TestRequire_ViewerCannotInviteMembers(t *testing.T) {
	repo := &stubMemberRepo{members: map[string]*domain.Membership{
		"u3:t1": activeMember("u3", "t1", domain.MEMBER_ROLE_VIEWER),
	}}
	checker := permission.New(repo)
	ctx := ctxWith("u3", "t1")

	if err := checker.Require(ctx, permission.MemberInvite); err == nil {
		t.Error("VIEWER should NOT have member:invite")
	}
	if err := checker.Require(ctx, permission.TenantView); err != nil {
		t.Errorf("VIEWER should have tenant:view, got: %v", err)
	}
}

func TestRequire_RevokedMembershipDenied(t *testing.T) {
	revoked := activeMember("u4", "t1", domain.MEMBER_ROLE_ADMIN)
	revoked.Status = domain.MEMBER_STATUS_REVOKED
	repo := &stubMemberRepo{members: map[string]*domain.Membership{"u4:t1": revoked}}
	checker := permission.New(repo)
	ctx := ctxWith("u4", "t1")

	err := checker.Require(ctx, permission.TenantView)
	if err == nil {
		t.Error("revoked membership should be denied")
	}
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.HTTPCode != 403 {
		t.Errorf("expected HTTP 403 AppError, got %v", err)
	}
}

func TestRequire_PlatformAdminBypassesAll(t *testing.T) {
	repo := &stubMemberRepo{members: map[string]*domain.Membership{}}
	checker := permission.New(repo)
	ctx := ctxPlatformAdmin()

	for _, perm := range []permission.Permission{
		permission.TenantDelete, permission.MemberRevoke, permission.AuditView,
	} {
		if err := checker.Require(ctx, perm); err != nil {
			t.Errorf("platform admin should bypass %s, got error: %v", perm, err)
		}
	}
}

func TestRequire_MissingContextIdentity(t *testing.T) {
	repo := &stubMemberRepo{members: map[string]*domain.Membership{}}
	checker := permission.New(repo)
	ctx := context.Background() // no userID/tenantID

	err := checker.Require(ctx, permission.TenantView)
	if err == nil {
		t.Error("missing identity should return error")
	}
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.HTTPCode != 401 {
		t.Errorf("expected HTTP 401 AppError, got %v", err)
	}
}

func TestRequire_MembershipNotFound(t *testing.T) {
	repo := &stubMemberRepo{members: map[string]*domain.Membership{}}
	checker := permission.New(repo)
	ctx := ctxWith("unknown", "t1")

	err := checker.Require(ctx, permission.TenantView)
	if err == nil {
		t.Error("unknown membership should return error")
	}
	var appErr *apperrors.AppError
	if !errors.As(err, &appErr) || appErr.HTTPCode != 404 {
		t.Errorf("expected HTTP 404 AppError for missing membership, got %v", err)
	}
}

func TestRequire_AgentHasLimitedPermissions(t *testing.T) {
	repo := &stubMemberRepo{members: map[string]*domain.Membership{
		"u5:t1": activeMember("u5", "t1", domain.MEMBER_ROLE_AGENT),
	}}
	checker := permission.New(repo)
	ctx := ctxWith("u5", "t1")

	// Agent CAN do these
	for _, perm := range []permission.Permission{
		permission.TenantView, permission.MemberList,
		permission.UserProfileView, permission.UserProfileUpdateOwn,
		permission.StorageUpload,
	} {
		if err := checker.Require(ctx, perm); err != nil {
			t.Errorf("AGENT should have %s, got error: %v", perm, err)
		}
	}

	// Agent CANNOT do these
	for _, perm := range []permission.Permission{
		permission.TenantUpdate, permission.TenantDelete,
		permission.MemberInvite, permission.MemberRevoke,
		permission.AuditView, permission.StorageDelete,
	} {
		if err := checker.Require(ctx, perm); err == nil {
			t.Errorf("AGENT should NOT have %s", perm)
		}
	}
}
