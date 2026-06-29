package infrastructure

import (
	"context"

	"github.com/agile-factory/kernel/kernel/errors"
	"github.com/agile-factory/kernel/kernel/txctx"
	"github.com/agile-factory/kernel/platform/user/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresUserProfileRepository implements domain.UserProfileRepository.
type PostgresUserProfileRepository struct {
	db *pgxpool.Pool
}

// NewPostgresUserProfileRepository creates a new repository backed by pgxpool.
func NewPostgresUserProfileRepository(db *pgxpool.Pool) *PostgresUserProfileRepository {
	return &PostgresUserProfileRepository{db: db}
}

// FindByUserID fetches the profile row for userID.
// Returns errors.NotFound if no row exists.
func (r *PostgresUserProfileRepository) FindByUserID(ctx context.Context, userID string) (*domain.UserProfile, error) {
	row := txctx.DB(ctx, r.db).QueryRow(ctx, queryFindByUserID, userID)

	var p domain.UserProfile
	var displayName, avatarURL, bio *string
	err := row.Scan(
		&p.UserID,
		&displayName,
		&avatarURL,
		&bio,
		&p.Locale,
		&p.Timezone,
		&p.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.NotFound("user profile not found")
		}
		return nil, errors.WrapInternal(err)
	}
	if displayName != nil {
		p.DisplayName = *displayName
	}
	if avatarURL != nil {
		p.AvatarURL = *avatarURL
	}
	if bio != nil {
		p.Bio = *bio
	}
	return &p, nil
}

// Upsert inserts or updates the profile row.
func (r *PostgresUserProfileRepository) Upsert(ctx context.Context, p *domain.UserProfile) error {
	var displayName, avatarURL, bio *string
	if p.DisplayName != "" {
		displayName = &p.DisplayName
	}
	if p.AvatarURL != "" {
		avatarURL = &p.AvatarURL
	}
	if p.Bio != "" {
		bio = &p.Bio
	}
	_, err := txctx.DB(ctx, r.db).Exec(ctx, queryUpsert,
		p.UserID,
		displayName,
		avatarURL,
		bio,
		p.Locale,
		p.Timezone,
		p.UpdatedAt,
	)
	if err != nil {
		return errors.WrapInternal(err)
	}
	return nil
}
