package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/buithean2010/mail-tracker/backend/internal/domain"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	const q = `SELECT id, microsoft_id, email, display_name, filter_settings, needs_reauth, last_synced_at, created_at, updated_at
               FROM users WHERE id = $1`
	return r.scan(r.pool.QueryRow(ctx, q, id))
}

func (r *UserRepo) GetByMicrosoftID(ctx context.Context, msID string) (*domain.User, error) {
	const q = `SELECT id, microsoft_id, email, display_name, filter_settings, needs_reauth, last_synced_at, created_at, updated_at
               FROM users WHERE microsoft_id = $1`
	return r.scan(r.pool.QueryRow(ctx, q, msID))
}

func (r *UserRepo) Upsert(ctx context.Context, user *domain.User) error {
	const q = `INSERT INTO users (microsoft_id, email, display_name, filter_settings)
               VALUES ($1, $2, $3, $4)
               ON CONFLICT (microsoft_id) DO UPDATE
               SET email = EXCLUDED.email, display_name = EXCLUDED.display_name, updated_at = NOW()`
	_, err := r.pool.Exec(ctx, q, user.MicrosoftID, user.Email, user.DisplayName, user.FilterSettings)
	if err != nil {
		return fmt.Errorf("upsert user: %w", err)
	}
	return nil
}

func (r *UserRepo) UpdateFilterSettings(ctx context.Context, userID uuid.UUID, settings json.RawMessage) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET filter_settings = $1, updated_at = NOW() WHERE id = $2`,
		settings, userID)
	return err
}

func (r *UserRepo) UpdateDisplayName(ctx context.Context, userID uuid.UUID, name string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET display_name = $1, updated_at = NOW() WHERE id = $2`,
		name, userID)
	return err
}

func (r *UserRepo) SetNeedsReauth(ctx context.Context, userID uuid.UUID, flag bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET needs_reauth = $1, updated_at = NOW() WHERE id = $2`,
		flag, userID)
	return err
}

func (r *UserRepo) SetLastSyncedAt(ctx context.Context, userID uuid.UUID, t time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET last_synced_at = $1, updated_at = NOW() WHERE id = $2`,
		t.UTC(), userID)
	return err
}

func (r *UserRepo) scan(row pgx.Row) (*domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.MicrosoftID, &u.Email, &u.DisplayName,
		&u.FilterSettings, &u.NeedsReauth, &u.LastSyncedAt, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
