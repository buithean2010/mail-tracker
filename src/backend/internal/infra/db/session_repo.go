package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/buithean2010/mail-tracker/backend/internal/domain"
)

type SessionRepo struct {
	pool *pgxpool.Pool
}

func NewSessionRepo(pool *pgxpool.Pool) *SessionRepo {
	return &SessionRepo{pool: pool}
}

func (r *SessionRepo) Create(ctx context.Context, s *domain.Session) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO sessions (id, user_id, expires_at) VALUES ($1, $2, $3)`,
		s.ID, s.UserID, s.ExpiresAt)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

func (r *SessionRepo) GetByID(ctx context.Context, id string) (*domain.Session, error) {
	var s domain.Session
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, expires_at, created_at FROM sessions WHERE id = $1`, id).
		Scan(&s.ID, &s.UserID, &s.ExpiresAt, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if s.ExpiresAt.Before(time.Now()) {
		return nil, domain.ErrNotFound
	}
	return &s, nil
}

func (r *SessionRepo) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE id = $1`, id)
	return err
}

func (r *SessionRepo) DeleteExpired(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE expires_at < NOW()`)
	return err
}

func (r *SessionRepo) GetActiveUsers(ctx context.Context, withinDays int) ([]*domain.User, error) {
	const q = `SELECT DISTINCT u.id, u.microsoft_id, u.email, u.display_name, u.filter_settings, u.needs_reauth, u.created_at, u.updated_at
               FROM users u
               JOIN sessions s ON s.user_id = u.id
               WHERE s.expires_at > NOW()
               AND s.created_at > NOW() - make_interval(days => $1)
               AND u.needs_reauth = false`
	rows, err := r.pool.Query(ctx, q, withinDays)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.MicrosoftID, &u.Email, &u.DisplayName,
			&u.FilterSettings, &u.NeedsReauth, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, rows.Err()
}
