package infrastructure

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	kerrors "github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/kernel/txctx"
	"github.com/agile-factory/kernel/platform/end_customers/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresEndCustomerRepository implements domain.EndCustomerRepository.
type PostgresEndCustomerRepository struct {
	db *pgxpool.Pool
}

// NewPostgresEndCustomerRepository creates a new repository backed by pgxpool.
func NewPostgresEndCustomerRepository(db *pgxpool.Pool) *PostgresEndCustomerRepository {
	return &PostgresEndCustomerRepository{db: db}
}

// ── cursor helpers ────────────────────────────────────────────────────────────

// listCursor encodes the keyset position for end_customers pagination.
// Canonical order: created_at DESC, id ASC.
type listCursor struct {
	CreatedAt time.Time `json:"created_at"`
	ID        string    `json:"id"`
}

func encodeCursor(c listCursor) (string, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("cursor encode: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func decodeCursor(s string) (*listCursor, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("cursor decode (base64): %w", err)
	}
	var c listCursor
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("cursor decode (json): %w", err)
	}
	if c.ID == "" {
		return nil, fmt.Errorf("cursor decode: missing id field")
	}
	return &c, nil
}

// ── scan helpers ──────────────────────────────────────────────────────────────

// scanEndCustomer reads one row from the canonical SELECT column list:
// id, tenant_id, external_id, email, phone, first_name, last_name,
// status, metadata, created_at, updated_at, deleted_at
func scanEndCustomer(row pgx.Row) (*domain.EndCustomer, error) {
	var ec domain.EndCustomer
	var externalID, email, phone *string
	var metaRaw []byte

	err := row.Scan(
		&ec.ID,
		&ec.TenantID,
		&externalID,
		&email,
		&phone,
		&ec.FirstName,
		&ec.LastName,
		&ec.Status,
		&metaRaw,
		&ec.CreatedAt,
		&ec.UpdatedAt,
		&ec.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrEndCustomerNotFound
		}
		return nil, kerrors.WrapInternal(err)
	}
	if externalID != nil {
		ec.ExternalID = *externalID
	}
	if email != nil {
		ec.Email = *email
	}
	if phone != nil {
		ec.Phone = *phone
	}
	if len(metaRaw) > 0 {
		if err := json.Unmarshal(metaRaw, &ec.Metadata); err != nil {
			return nil, kerrors.WrapInternal(fmt.Errorf("metadata unmarshal: %w", err))
		}
	}
	if ec.Metadata == nil {
		ec.Metadata = map[string]string{}
	}
	return &ec, nil
}

// loadTags fetches all tags for ec and attaches them to ec.Tags.
func (r *PostgresEndCustomerRepository) loadTags(ctx context.Context, ec *domain.EndCustomer) error {
	rows, err := txctx.DB(ctx, r.db).Query(ctx, queryFindTags, ec.ID)
	if err != nil {
		return kerrors.WrapInternal(err)
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return kerrors.WrapInternal(err)
		}
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		return kerrors.WrapInternal(err)
	}
	ec.Tags = tags
	return nil
}

// replaceTags deletes all existing tags for ec.ID and inserts the new set.
// Must be called inside a transaction (txctx).
func (r *PostgresEndCustomerRepository) replaceTags(ctx context.Context, ec *domain.EndCustomer) error {
	db := txctx.DB(ctx, r.db)
	if _, err := db.Exec(ctx, queryDeleteTags, ec.ID); err != nil {
		return kerrors.WrapInternal(err)
	}
	for _, tag := range ec.Tags {
		if _, err := db.Exec(ctx, queryInsertTag, ec.ID, tag); err != nil {
			return kerrors.WrapInternal(err)
		}
	}
	return nil
}

// isUniqueViolation returns true when err is a PostgreSQL unique-constraint
// violation (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// ── nullable args ─────────────────────────────────────────────────────────────

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ── Create ────────────────────────────────────────────────────────────────────

// Create inserts a new end-customer and its tags inside the same transaction.
func (r *PostgresEndCustomerRepository) Create(ctx context.Context, ec *domain.EndCustomer) error {
	metaRaw, err := json.Marshal(ec.Metadata)
	if err != nil {
		return kerrors.WrapInternal(fmt.Errorf("metadata marshal: %w", err))
	}

	db := txctx.DB(ctx, r.db)
	_, err = db.Exec(ctx, queryCreate,
		ec.ID,
		ec.TenantID,
		nullableString(ec.ExternalID),
		nullableString(ec.Email),
		nullableString(ec.Phone),
		ec.FirstName,
		ec.LastName,
		ec.Status,
		metaRaw,
		ec.CreatedAt,
		ec.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrEndCustomerConflict
		}
		return kerrors.WrapInternal(err)
	}
	return r.replaceTags(ctx, ec)
}

// ── FindByID ──────────────────────────────────────────────────────────────────

// FindByID returns the end-customer for the given tenant + id.
func (r *PostgresEndCustomerRepository) FindByID(ctx context.Context, tenantID, id string) (*domain.EndCustomer, error) {
	row := txctx.DB(ctx, r.db).QueryRow(ctx, queryFindByID, tenantID, id)
	ec, err := scanEndCustomer(row)
	if err != nil {
		return nil, err
	}
	if err := r.loadTags(ctx, ec); err != nil {
		return nil, err
	}
	return ec, nil
}

// ── FindByEmail ───────────────────────────────────────────────────────────────

// FindByEmail returns the end-customer matching email in tenant.
func (r *PostgresEndCustomerRepository) FindByEmail(ctx context.Context, tenantID, email string) (*domain.EndCustomer, error) {
	row := txctx.DB(ctx, r.db).QueryRow(ctx, queryFindByEmail, tenantID, email)
	ec, err := scanEndCustomer(row)
	if err != nil {
		return nil, err
	}
	if err := r.loadTags(ctx, ec); err != nil {
		return nil, err
	}
	return ec, nil
}

// ── FindByExternalID ──────────────────────────────────────────────────────────

// FindByExternalID returns the end-customer matching external_id in tenant.
func (r *PostgresEndCustomerRepository) FindByExternalID(ctx context.Context, tenantID, externalID string) (*domain.EndCustomer, error) {
	row := txctx.DB(ctx, r.db).QueryRow(ctx, queryFindByExternalID, tenantID, externalID)
	ec, err := scanEndCustomer(row)
	if err != nil {
		return nil, err
	}
	if err := r.loadTags(ctx, ec); err != nil {
		return nil, err
	}
	return ec, nil
}

// ── Update ────────────────────────────────────────────────────────────────────

// Update persists all mutable fields of ec and replaces its tags.
func (r *PostgresEndCustomerRepository) Update(ctx context.Context, ec *domain.EndCustomer) error {
	metaRaw, err := json.Marshal(ec.Metadata)
	if err != nil {
		return kerrors.WrapInternal(fmt.Errorf("metadata marshal: %w", err))
	}

	db := txctx.DB(ctx, r.db)
	tag, err := db.Exec(ctx, queryUpdate,
		ec.TenantID,
		ec.ID,
		nullableString(ec.ExternalID),
		nullableString(ec.Email),
		nullableString(ec.Phone),
		ec.FirstName,
		ec.LastName,
		ec.Status,
		metaRaw,
		ec.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrEndCustomerConflict
		}
		return kerrors.WrapInternal(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrEndCustomerNotFound
	}
	return r.replaceTags(ctx, ec)
}

// ── SoftDelete ────────────────────────────────────────────────────────────────

// SoftDelete sets deleted_at = NOW() and status = DELETED.
func (r *PostgresEndCustomerRepository) SoftDelete(ctx context.Context, tenantID, id string) error {
	tag, err := txctx.DB(ctx, r.db).Exec(ctx, querySoftDelete, tenantID, id)
	if err != nil {
		return kerrors.WrapInternal(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrEndCustomerNotFound
	}
	return nil
}

// ── List ──────────────────────────────────────────────────────────────────────

// List returns a page of end-customers using keyset (cursor) pagination.
// The canonical order is created_at DESC, id ASC for stable tie-breaking.
// Returns the next cursor token (empty when no more pages).
func (r *PostgresEndCustomerRepository) List(ctx context.Context, params domain.ListParams) ([]*domain.EndCustomer, string, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}

	// Decode cursor (nil = first page).
	var after *listCursor
	if params.Cursor != "" {
		c, err := decodeCursor(params.Cursor)
		if err != nil {
			return nil, "", kerrors.BadRequest("invalid cursor")
		}
		after = c
	}

	// Build dynamic query.
	var sb strings.Builder
	args := []any{params.TenantID}
	argN := 2 // next placeholder index

	needsTagJoin := params.Tag != ""

	if needsTagJoin {
		sb.WriteString(`
SELECT DISTINCT ec.id, ec.tenant_id, ec.external_id, ec.email, ec.phone,
       ec.first_name, ec.last_name, ec.status, ec.metadata,
       ec.created_at, ec.updated_at, ec.deleted_at
FROM end_customers ec
JOIN end_customer_tags t ON t.end_customer_id = ec.id
WHERE ec.tenant_id = $1 AND ec.deleted_at IS NULL
`)
	} else {
		sb.WriteString(`
SELECT id, tenant_id, external_id, email, phone,
       first_name, last_name, status, metadata,
       created_at, updated_at, deleted_at
FROM end_customers
WHERE tenant_id = $1 AND deleted_at IS NULL
`)
	}

	if params.Status != nil {
		sb.WriteString(fmt.Sprintf(" AND %sstatus = $%d", tableAlias(needsTagJoin, "ec."), argN))
		args = append(args, *params.Status)
		argN++
	}

	if params.Tag != "" {
		sb.WriteString(fmt.Sprintf(" AND t.tag = $%d", argN))
		args = append(args, params.Tag)
		argN++
	}

	if params.Search != "" {
		col := tableAlias(needsTagJoin, "ec.")
		sb.WriteString(fmt.Sprintf(
			" AND to_tsvector('simple', coalesce(%sfirst_name,'') || ' ' || coalesce(%slast_name,'') || ' ' || coalesce(%semail,'')) @@ plainto_tsquery('simple', $%d)",
			col, col, col, argN,
		))
		args = append(args, params.Search)
		argN++
	}

	if after != nil {
		col := tableAlias(needsTagJoin, "ec.")
		// Keyset: (created_at < cursor_ts) OR (created_at = cursor_ts AND id > cursor_id)
		sb.WriteString(fmt.Sprintf(
			" AND (%screated_at < $%d OR (%screated_at = $%d AND %sid > $%d))",
			col, argN, col, argN+1, col, argN+2,
		))
		args = append(args, after.CreatedAt, after.CreatedAt, after.ID)
		argN += 3
	}

	col := tableAlias(needsTagJoin, "ec.")
	sb.WriteString(fmt.Sprintf(" ORDER BY %screated_at DESC, %sid ASC LIMIT $%d", col, col, argN))
	args = append(args, limit+1) // fetch one extra to detect next page

	rows, err := txctx.DB(ctx, r.db).Query(ctx, sb.String(), args...)
	if err != nil {
		return nil, "", kerrors.WrapInternal(err)
	}
	defer rows.Close()

	var items []*domain.EndCustomer
	for rows.Next() {
		var ec domain.EndCustomer
		var externalID, email, phone *string
		var metaRaw []byte

		if err := rows.Scan(
			&ec.ID,
			&ec.TenantID,
			&externalID,
			&email,
			&phone,
			&ec.FirstName,
			&ec.LastName,
			&ec.Status,
			&metaRaw,
			&ec.CreatedAt,
			&ec.UpdatedAt,
			&ec.DeletedAt,
		); err != nil {
			return nil, "", kerrors.WrapInternal(err)
		}
		if externalID != nil {
			ec.ExternalID = *externalID
		}
		if email != nil {
			ec.Email = *email
		}
		if phone != nil {
			ec.Phone = *phone
		}
		if len(metaRaw) > 0 {
			if err := json.Unmarshal(metaRaw, &ec.Metadata); err != nil {
				return nil, "", kerrors.WrapInternal(fmt.Errorf("metadata unmarshal: %w", err))
			}
		}
		if ec.Metadata == nil {
			ec.Metadata = map[string]string{}
		}
		items = append(items, &ec)
	}
	if err := rows.Err(); err != nil {
		return nil, "", kerrors.WrapInternal(err)
	}

	// Detect next page.
	var nextCursor string
	if len(items) > limit {
		items = items[:limit]
		last := items[len(items)-1]
		c, err := encodeCursor(listCursor{CreatedAt: last.CreatedAt, ID: last.ID})
		if err != nil {
			return nil, "", kerrors.WrapInternal(err)
		}
		nextCursor = c
	}

	// Load tags for each item.
	for _, ec := range items {
		if err := r.loadTags(ctx, ec); err != nil {
			return nil, "", err
		}
	}

	return items, nextCursor, nil
}

// tableAlias returns either the alias prefix (e.g. "ec.") or an empty string
// depending on whether a JOIN alias is in use.
func tableAlias(useAlias bool, alias string) string {
	if useAlias {
		return alias
	}
	return ""
}

// ── Count ─────────────────────────────────────────────────────────────────────

// Count returns the total number of non-deleted end-customers for the tenant.
// Optionally filtered by status.
func (r *PostgresEndCustomerRepository) Count(ctx context.Context, tenantID string, status *domain.EndCustomerStatus) (int64, error) {
	var n int64
	var err error
	if status == nil {
		err = txctx.DB(ctx, r.db).QueryRow(ctx, queryCount, tenantID).Scan(&n)
	} else {
		err = txctx.DB(ctx, r.db).QueryRow(ctx, queryCountByStatus, tenantID, *status).Scan(&n)
	}
	if err != nil {
		return 0, kerrors.WrapInternal(err)
	}
	return n, nil
}
