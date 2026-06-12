package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/buithean2010/mail-tracker/backend/internal/domain"
)

type TokenRepo struct {
	pool *pgxpool.Pool
	enc  domain.Encryptor
}

func NewTokenRepo(pool *pgxpool.Pool, enc domain.Encryptor) *TokenRepo {
	return &TokenRepo{pool: pool, enc: enc}
}

func (r *TokenRepo) Upsert(ctx context.Context, t *domain.UserToken) error {
	encAT, err := r.enc.Encrypt(t.AccessToken)
	if err != nil {
		return fmt.Errorf("encrypt access token: %w", err)
	}
	encRT, err := r.enc.Encrypt(t.RefreshToken)
	if err != nil {
		return fmt.Errorf("encrypt refresh token: %w", err)
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO user_tokens (user_id, access_token, refresh_token, expires_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE
		SET access_token = EXCLUDED.access_token,
		    refresh_token = EXCLUDED.refresh_token,
		    expires_at = EXCLUDED.expires_at,
		    updated_at = NOW()`,
		t.UserID, encAT, encRT, t.ExpiresAt)
	return err
}

func (r *TokenRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.UserToken, error) {
	var t domain.UserToken
	var encAT, encRT string
	err := r.pool.QueryRow(ctx,
		`SELECT user_id, access_token, refresh_token, expires_at, updated_at FROM user_tokens WHERE user_id = $1`,
		userID).Scan(&t.UserID, &encAT, &encRT, &t.ExpiresAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	t.AccessToken, err = r.enc.Decrypt(encAT)
	if err != nil {
		return nil, fmt.Errorf("decrypt access token: %w", err)
	}
	t.RefreshToken, err = r.enc.Decrypt(encRT)
	if err != nil {
		return nil, fmt.Errorf("decrypt refresh token: %w", err)
	}

	return &t, nil
}

func (r *TokenRepo) Delete(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM user_tokens WHERE user_id = $1`, userID)
	return err
}
