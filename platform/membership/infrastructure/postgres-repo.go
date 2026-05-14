package infrastructure

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Cheikh-Nakamoto/Agile-Factory-Boilerplate/backend/internal/platform/membership/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// pgUniqueViolation is the SQLSTATE code Postgres returns on UNIQUE conflict.
const pgUniqueViolation = "23505"

// PostgresMembershipRepository implements domain.MembershipRepository using pgxpool.
type PostgresMembershipRepository struct {
	mu sync.RWMutex
	db *pgxpool.Pool
}

// NewPostgresMembershipRepository creates a new PostgresMembershipRepository.
func NewPostgresMembershipRepository(db *pgxpool.Pool) *PostgresMembershipRepository {
	return &PostgresMembershipRepository{db: db}
}

// Save persists a new membership row.
func (r *PostgresMembershipRepository) Save(ctx context.Context, m *domain.Membership) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var invitedBy interface{}
	if m.InvitedBy != "" {
		invitedBy = m.InvitedBy
	}

	_, err := r.db.Exec(ctx, queryInsertMembership,
		m.ID,
		m.UserID,
		m.TenantID,
		string(m.Role),
		string(m.Status),
		invitedBy,
		m.JoinedAt,
		m.CreatedAt,
		m.UpdatedAt,
	)
	// Translate the UNIQUE(user_id, tenant_id) violation that wins the race
	// between concurrent invites into the domain-level "already exists" error.
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		return domain.ErrMembershipAlreadyExists
	}
	return err
}

// FindByID retrieves a membership by its UUID.
func (r *PostgresMembershipRepository) FindByID(ctx context.Context, id string) (*domain.Membership, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.scanOne(r.db.QueryRow(ctx, queryGetMembershipByID, id))
}

// FindByUserAndTenant retrieves the unique membership for a user/tenant pair.
func (r *PostgresMembershipRepository) FindByUserAndTenant(ctx context.Context, userID, tenantID string) (*domain.Membership, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.scanOne(r.db.QueryRow(ctx, queryGetMembershipByUserAndTenant, userID, tenantID))
}

// ListByTenant returns all memberships for a tenant.
func (r *PostgresMembershipRepository) ListByTenant(ctx context.Context, tenantID string) ([]*domain.Membership, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.scanMany(ctx, queryListMembershipsByTenant, tenantID)
}

// ListByUser returns all memberships a user holds across tenants.
func (r *PostgresMembershipRepository) ListByUser(ctx context.Context, userID string) ([]*domain.Membership, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.scanMany(ctx, queryListMembershipsByUser, userID)
}

// UpdateStatus persists a status and optional joined_at change.
func (r *PostgresMembershipRepository) UpdateStatus(ctx context.Context, id string, status domain.MemberStatus, joinedAt *time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, err := r.db.Exec(ctx, queryUpdateMembershipStatus, id, string(status), joinedAt)
	return err
}

// ─── internal helpers ────────────────────────────────────────────────────────

func (r *PostgresMembershipRepository) scanOne(row pgx.Row) (*domain.Membership, error) {
	m, err := scanMembership(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrMembershipNotFound
		}
		return nil, err
	}
	return m, nil
}

func (r *PostgresMembershipRepository) scanMany(ctx context.Context, query string, arg string) ([]*domain.Membership, error) {
	rows, err := r.db.Query(ctx, query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*domain.Membership
	for rows.Next() {
		m, err := scanMembership(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// scanMembership reads one row (pgx.Row or pgx.Rows both satisfy the scanner interface).
func scanMembership(scanner interface {
	Scan(dest ...any) error
}) (*domain.Membership, error) {
	var m domain.Membership
	var role, status string
	var invitedBy string

	err := scanner.Scan(
		&m.ID,
		&m.UserID,
		&m.TenantID,
		&role,
		&status,
		&invitedBy,
		&m.JoinedAt,
		&m.CreatedAt,
		&m.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	m.Role = domain.MemberRole(role)
	m.Status = domain.MemberStatus(status)
	m.InvitedBy = invitedBy
	return &m, nil
}
