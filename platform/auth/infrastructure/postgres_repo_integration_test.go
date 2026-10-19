package infrastructure_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/agile-factory/kernel/kernel/events"
	"github.com/agile-factory/kernel/kernel/migrator"
	"github.com/agile-factory/kernel/platform/auth"
	"github.com/agile-factory/kernel/platform/auth/domain"
	authinfra "github.com/agile-factory/kernel/platform/auth/infrastructure"
	"github.com/agile-factory/kernel/platform/tenant"
	tenantdomain "github.com/agile-factory/kernel/platform/tenant/domain"
	tenantinfra "github.com/agile-factory/kernel/platform/tenant/infrastructure"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// KERNEL_TEST_DATABASE_URL names a throwaway Postgres (CI sets it); the test
// is skipped without it.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("KERNEL_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("KERNEL_TEST_DATABASE_URL not set")
	}
	if err := migrator.Run(dsn, []migrator.Source{
		{Name: "events", FS: events.Migrations, Path: "migrations"},
		{Name: "tenant", FS: tenant.Migrations, Path: "migrations"},
		{Name: "auth", FS: auth.Migrations, Path: "migrations"},
	}); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// Regression: FindByEmail / FindByID scanned the email column straight into
// domain.Email, which pgx cannot do, so every login failed.
func TestPostgresUserRepository_RoundTrip(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	repo := authinfra.NewPostgresUserRepository(pool)

	email, err := domain.NewEmail(fmt.Sprintf("User-%d@Example.org", time.Now().UnixNano()))
	if err != nil {
		t.Fatal(err)
	}
	u := domain.NewUser(uuid.NewString(), email, "$2a$10$hash", domain.ROLE_VIEWER)
	if err := repo.Save(ctx, u); err != nil {
		t.Fatal(err)
	}
	byEmail, err := repo.FindByEmail(ctx, email)
	if err != nil {
		t.Fatalf("FindByEmail: %v", err)
	}
	byID, err := repo.FindByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	for _, got := range []*domain.User{byEmail, byID} {
		if got.ID != u.ID || got.Email != email || got.Status != u.Status || got.HashedPassword != u.HashedPassword {
			t.Errorf("round trip mismatch: got %+v, want %+v", got, u)
		}
	}
}

// Regression: tenants.name was UNIQUE, so a second tenant with the same
// display name failed (a 500 at signup). Slugs stay unique.
func TestTenants_NamesMayRepeatSlugsMayNot(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	repo := tenantinfra.NewPostgresTenantRepository(pool)
	run := time.Now().UnixNano()

	for i := 0; i < 2; i++ {
		tn, err := tenantdomain.NewTenant(uuid.NewString(), "Maison Diallo", fmt.Sprintf("diallo-%d-%d", run, i))
		if err != nil {
			t.Fatal(err)
		}
		if err := repo.Save(ctx, tn); err != nil {
			t.Fatalf("tenant %d with a repeated name: %v", i, err)
		}
	}
	dup, _ := tenantdomain.NewTenant(uuid.NewString(), "Other", fmt.Sprintf("diallo-%d-0", run))
	if err := repo.Save(ctx, dup); err == nil {
		t.Fatal("a duplicate slug was accepted")
	}
}
