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

type APIKeyRepo struct {
	pool *pgxpool.Pool
	enc  domain.Encryptor
}

func NewAPIKeyRepo(pool *pgxpool.Pool, enc domain.Encryptor) *APIKeyRepo {
	return &APIKeyRepo{pool: pool, enc: enc}
}

func (r *APIKeyRepo) Upsert(ctx context.Context, k *domain.UserAPIKey) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_api_keys (user_id, ai_mode, provider, api_key, model, pa_webhook_url)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (user_id) DO UPDATE
		SET ai_mode=$2, provider=$3, api_key=$4, model=$5, pa_webhook_url=$6, updated_at=NOW()`,
		k.UserID, k.AIMode, k.Provider, k.APIKey, k.Model, k.PAWebhookURL)
	return err
}

func (r *APIKeyRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.UserAPIKey, error) {
	var k domain.UserAPIKey
	var encKey, encWebhook string
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, ai_mode, provider, api_key, model, pa_webhook_url, created_at, updated_at
		FROM user_api_keys WHERE user_id = $1`, userID).
		Scan(&k.ID, &k.UserID, &k.AIMode, &k.Provider, &encKey, &k.Model, &encWebhook, &k.CreatedAt, &k.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if encKey != "" {
		k.APIKey, err = r.enc.Decrypt(encKey)
		if err != nil {
			return nil, fmt.Errorf("decrypt api key: %w", err)
		}
	}
	if encWebhook != "" {
		k.PAWebhookURL, err = r.enc.Decrypt(encWebhook)
		if err != nil {
			return nil, fmt.Errorf("decrypt webhook url: %w", err)
		}
	}

	return &k, nil
}

func (r *APIKeyRepo) Delete(ctx context.Context, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM user_api_keys WHERE user_id = $1`, userID)
	return err
}
