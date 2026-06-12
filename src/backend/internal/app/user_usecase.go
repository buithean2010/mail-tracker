package app

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/buithean2010/mail-tracker/backend/internal/domain"
)

type UserUseCase struct {
	userRepo   domain.UserRepo
	apiKeyRepo domain.APIKeyRepo
	enc        domain.Encryptor
}

func NewUserUseCase(userRepo domain.UserRepo, apiKeyRepo domain.APIKeyRepo, enc domain.Encryptor) *UserUseCase {
	return &UserUseCase{userRepo: userRepo, apiKeyRepo: apiKeyRepo, enc: enc}
}

func (uc *UserUseCase) GetProfile(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	return uc.userRepo.GetByID(ctx, userID)
}

func (uc *UserUseCase) UpdateDisplayName(ctx context.Context, userID uuid.UUID, name string) error {
	if name == "" {
		return domain.ErrInvalidInput
	}
	return uc.userRepo.UpdateDisplayName(ctx, userID, name)
}

func (uc *UserUseCase) GetFilter(ctx context.Context, userID uuid.UUID) (*domain.FilterSettings, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return domain.ParseFilterSettings(user.FilterSettings)
}

func (uc *UserUseCase) UpdateFilter(ctx context.Context, userID uuid.UUID, settings *domain.FilterSettings) error {
	raw, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("marshal filter: %w", err)
	}
	return uc.userRepo.UpdateFilterSettings(ctx, userID, raw)
}

func (uc *UserUseCase) GetAPIKey(ctx context.Context, userID uuid.UUID) (*domain.UserAPIKey, error) {
	key, err := uc.apiKeyRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	// Mask sensitive fields for API response
	key.APIKey = ""
	key.PAWebhookURL = ""
	return key, nil
}

func (uc *UserUseCase) UpsertAPIKey(ctx context.Context, userID uuid.UUID, key *domain.UserAPIKey) error {
	key.UserID = userID

	// Encrypt sensitive fields before storage
	if key.AIMode == domain.AIModeByok && key.APIKey != "" {
		enc, err := uc.enc.Encrypt(key.APIKey)
		if err != nil {
			return fmt.Errorf("encrypt api key: %w", err)
		}
		key.APIKey = enc
	}
	if key.AIMode == domain.AIModePA && key.PAWebhookURL != "" {
		enc, err := uc.enc.Encrypt(key.PAWebhookURL)
		if err != nil {
			return fmt.Errorf("encrypt webhook url: %w", err)
		}
		key.PAWebhookURL = enc
	}

	return uc.apiKeyRepo.Upsert(ctx, key)
}

func (uc *UserUseCase) DeleteAPIKey(ctx context.Context, userID uuid.UUID) error {
	return uc.apiKeyRepo.Delete(ctx, userID)
}
