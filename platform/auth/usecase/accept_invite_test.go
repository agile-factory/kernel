package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	authdomain "github.com/agile-factory/kernel/platform/auth/domain"
	"github.com/agile-factory/kernel/platform/auth/usecase"
	memberdomain "github.com/agile-factory/kernel/platform/membership/domain"
)

// ─── Mocks ───────────────────────────────────────────────────────────────────

type mockHasher struct{ mockVerifier }

func (m *mockHasher) Hash(plain string) (string, error) { return "hashed_" + plain, nil }

type mockInviteTokenRepo struct {
	invite     *memberdomain.InviteToken
	consumedID string
}

func (m *mockInviteTokenRepo) Save(context.Context, *memberdomain.InviteToken) error { return nil }
func (m *mockInviteTokenRepo) FindByToken(_ context.Context, token string) (*memberdomain.InviteToken, error) {
	if m.invite == nil || m.invite.Token != token {
		return nil, memberdomain.ErrInviteTokenNotFound
	}
	return m.invite, nil
}
func (m *mockInviteTokenRepo) FindPendingByTenantAndEmail(context.Context, string, string) (*memberdomain.InviteToken, error) {
	return nil, memberdomain.ErrInviteTokenNotFound
}
func (m *mockInviteTokenRepo) Consume(_ context.Context, id string) error {
	m.consumedID = id
	return nil
}
func (m *mockInviteTokenRepo) Revoke(context.Context, string) error { return nil }

type mockMembershipRepo struct {
	saved []*memberdomain.Membership
}

func (m *mockMembershipRepo) Save(_ context.Context, ms *memberdomain.Membership) error {
	m.saved = append(m.saved, ms)
	return nil
}
func (m *mockMembershipRepo) FindByID(context.Context, string) (*memberdomain.Membership, error) {
	return nil, memberdomain.ErrMembershipNotFound
}
func (m *mockMembershipRepo) FindByUserAndTenant(context.Context, string, string) (*memberdomain.Membership, error) {
	return nil, memberdomain.ErrMembershipNotFound
}
func (m *mockMembershipRepo) ListByTenant(context.Context, string) ([]*memberdomain.Membership, error) {
	return nil, nil
}
func (m *mockMembershipRepo) ListByUser(context.Context, string) ([]*memberdomain.Membership, error) {
	return nil, nil
}
func (m *mockMembershipRepo) UpdateStatus(context.Context, string, memberdomain.MemberStatus, *time.Time) error {
	return nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

const inviteRaw = "raw-invite-token"

func pendingInvite(t *testing.T, email string) *memberdomain.InviteToken {
	t.Helper()
	inv, err := memberdomain.NewInviteToken("inv-1", "tenant-1", "inviter-1", email, memberdomain.MEMBER_ROLE_AGENT, inviteRaw, time.Hour)
	if err != nil {
		t.Fatalf("NewInviteToken: %v", err)
	}
	return inv
}

type acceptFixture struct {
	users       *mockUserRepo
	invites     *mockInviteTokenRepo
	memberships *mockMembershipRepo
	refresh     *mockRefreshTokenRepo
	uc          *usecase.AcceptInviteUseCase
}

func newAcceptFixture(t *testing.T, users *mockUserRepo, email string) *acceptFixture {
	t.Helper()
	f := &acceptFixture{
		users:       users,
		invites:     &mockInviteTokenRepo{invite: pendingInvite(t, email)},
		memberships: &mockMembershipRepo{},
		refresh:     &mockRefreshTokenRepo{},
	}
	f.uc = usecase.NewAcceptInviteUseCase(
		nil, f.users, f.memberships, f.invites, f.refresh,
		&mockHasher{}, &mockTokenGenerator{}, nil, nil, time.Hour,
	).WithoutTx()
	return f
}

func existingUserRepo(u *authdomain.User) *mockUserRepo {
	return &mockUserRepo{
		findByEmailFn: func(context.Context, authdomain.Email) (*authdomain.User, error) { return u, nil },
	}
}

// ─── Tests ───────────────────────────────────────────────────────────────────

func TestAcceptInvite_ExistingUser_WithoutPassword_Rejected(t *testing.T) {
	victim := activeUser(t, "victim", "victim@agile.io")
	f := newAcceptFixture(t, existingUserRepo(victim), "victim@agile.io")

	_, err := f.uc.Execute(context.Background(), inviteRaw, "")
	if !errors.Is(err, authdomain.ErrInvalidCredentials) {
		t.Fatalf("want ErrInvalidCredentials, got %v", err)
	}
	if f.invites.consumedID != "" || len(f.memberships.saved) != 0 || len(f.refresh.saved) != 0 {
		t.Fatal("rejected accept must not consume the invite, create a membership or issue tokens")
	}
}

func TestAcceptInvite_ExistingUser_WrongPassword_Rejected(t *testing.T) {
	victim := activeUser(t, "victim", "victim@agile.io")
	f := newAcceptFixture(t, existingUserRepo(victim), "victim@agile.io")

	_, err := f.uc.Execute(context.Background(), inviteRaw, "attacker-guess-123")
	if !errors.Is(err, authdomain.ErrInvalidCredentials) {
		t.Fatalf("want ErrInvalidCredentials, got %v", err)
	}
}

func TestAcceptInvite_ExistingUser_CorrectPassword_Joins(t *testing.T) {
	u := activeUser(t, "user-1", "bob@agile.io")
	f := newAcceptFixture(t, existingUserRepo(u), "bob@agile.io")

	out, err := f.uc.Execute(context.Background(), inviteRaw, "Password123!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.User.ID != "user-1" || out.Membership.TenantID != "tenant-1" {
		t.Fatalf("unexpected output: user=%s tenant=%s", out.User.ID, out.Membership.TenantID)
	}
	if out.AccessToken == "" || out.RefreshToken == "" {
		t.Fatal("expected a token pair")
	}
	if f.invites.consumedID != "inv-1" {
		t.Fatal("invite should be consumed")
	}
}

func TestAcceptInvite_ExistingUser_Suspended_Rejected(t *testing.T) {
	u := activeUser(t, "user-1", "bob@agile.io")
	u.Status = authdomain.USER_STATUS_SUSPENDED
	f := newAcceptFixture(t, existingUserRepo(u), "bob@agile.io")

	_, err := f.uc.Execute(context.Background(), inviteRaw, "Password123!")
	if !errors.Is(err, authdomain.ErrUserNotActive) {
		t.Fatalf("want ErrUserNotActive, got %v", err)
	}
}

func TestAcceptInvite_LookupError_NotTreatedAsNewUser(t *testing.T) {
	saved := false
	users := &mockUserRepo{
		findByEmailFn: func(context.Context, authdomain.Email) (*authdomain.User, error) {
			return nil, errors.New("connection reset")
		},
		saveFn: func(context.Context, *authdomain.User) error { saved = true; return nil },
	}
	f := newAcceptFixture(t, users, "new@agile.io")

	if _, err := f.uc.Execute(context.Background(), inviteRaw, "Password123!"); err == nil {
		t.Fatal("expected an error")
	}
	if saved {
		t.Fatal("a repository error must not lead to creating a user")
	}
}

func TestAcceptInvite_NewUser_Created(t *testing.T) {
	var created *authdomain.User
	users := &mockUserRepo{
		saveFn: func(_ context.Context, u *authdomain.User) error { created = u; return nil },
	}
	f := newAcceptFixture(t, users, "new@agile.io")

	out, err := f.uc.Execute(context.Background(), inviteRaw, "Password123!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created == nil || created.HashedPassword != "hashed_Password123!" {
		t.Fatal("expected the new user to be saved with the hashed password")
	}
	if out.User != created {
		t.Fatal("output should carry the created user")
	}
}

func TestAcceptInvite_NewUser_ShortPassword_Rejected(t *testing.T) {
	f := newAcceptFixture(t, &mockUserRepo{}, "new@agile.io")

	_, err := f.uc.Execute(context.Background(), inviteRaw, "short")
	if !errors.Is(err, authdomain.ErrPasswordTooShort) {
		t.Fatalf("want ErrPasswordTooShort, got %v", err)
	}
}
