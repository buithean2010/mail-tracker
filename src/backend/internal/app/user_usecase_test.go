package app_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/buithean2010/mail-tracker/backend/internal/app"
	"github.com/buithean2010/mail-tracker/backend/internal/domain"
)

// mockUserRepo implements domain.UserRepo for testing.
type mockUserRepo struct {
	users            map[uuid.UUID]*domain.User
	filterSettingSet json.RawMessage
	displayNameSet   string
	needsReauthSet   bool
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[uuid.UUID]*domain.User)}
}

func (m *mockUserRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}

func (m *mockUserRepo) GetByMicrosoftID(_ context.Context, msID string) (*domain.User, error) {
	for _, u := range m.users {
		if u.MicrosoftID == msID {
			return u, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *mockUserRepo) Upsert(_ context.Context, u *domain.User) error {
	m.users[u.ID] = u
	return nil
}

func (m *mockUserRepo) UpdateFilterSettings(_ context.Context, _ uuid.UUID, settings json.RawMessage) error {
	m.filterSettingSet = settings
	return nil
}

func (m *mockUserRepo) UpdateDisplayName(_ context.Context, _ uuid.UUID, name string) error {
	m.displayNameSet = name
	return nil
}

func (m *mockUserRepo) SetNeedsReauth(_ context.Context, _ uuid.UUID, flag bool) error {
	m.needsReauthSet = flag
	return nil
}

// mockAPIKeyRepo implements domain.APIKeyRepo for testing.
type mockAPIKeyRepo struct {
	keys   map[uuid.UUID]*domain.UserAPIKey
	upsert *domain.UserAPIKey
}

func newMockAPIKeyRepo() *mockAPIKeyRepo {
	return &mockAPIKeyRepo{keys: make(map[uuid.UUID]*domain.UserAPIKey)}
}

func (m *mockAPIKeyRepo) Upsert(_ context.Context, k *domain.UserAPIKey) error {
	m.upsert = k
	m.keys[k.UserID] = k
	return nil
}

func (m *mockAPIKeyRepo) GetByUserID(_ context.Context, userID uuid.UUID) (*domain.UserAPIKey, error) {
	k, ok := m.keys[userID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *k
	return &cp, nil
}

func (m *mockAPIKeyRepo) Delete(_ context.Context, userID uuid.UUID) error {
	delete(m.keys, userID)
	return nil
}

// mockEncryptor is a trivially reversible encryptor for testing.
type mockEncryptor struct{}

func (e *mockEncryptor) Encrypt(plaintext string) (string, error) { return "enc:" + plaintext, nil }
func (e *mockEncryptor) Decrypt(ciphertext string) (string, error) {
	if len(ciphertext) < 4 {
		return ciphertext, nil
	}
	return ciphertext[4:], nil
}

// --- UserUseCase tests ---

func newUserUC() (*app.UserUseCase, *mockUserRepo, *mockAPIKeyRepo) {
	ur := newMockUserRepo()
	ar := newMockAPIKeyRepo()
	enc := &mockEncryptor{}
	return app.NewUserUseCase(ur, ar, enc), ur, ar
}

func TestUserUseCase_GetProfile(t *testing.T) {
	uc, ur, _ := newUserUC()
	userID := uuid.New()
	ur.users[userID] = &domain.User{ID: userID, Email: "me@example.com"}

	u, err := uc.GetProfile(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Email != "me@example.com" {
		t.Errorf("email = %q, want me@example.com", u.Email)
	}
}

func TestUserUseCase_GetProfile_NotFound(t *testing.T) {
	uc, _, _ := newUserUC()
	_, err := uc.GetProfile(context.Background(), uuid.New())
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUserUseCase_UpdateDisplayName(t *testing.T) {
	uc, ur, _ := newUserUC()
	userID := uuid.New()

	if err := uc.UpdateDisplayName(context.Background(), userID, "Bob"); err != nil {
		t.Fatal(err)
	}
	if ur.displayNameSet != "Bob" {
		t.Errorf("displayName = %q, want Bob", ur.displayNameSet)
	}
}

func TestUserUseCase_UpdateDisplayName_Empty(t *testing.T) {
	uc, _, _ := newUserUC()
	err := uc.UpdateDisplayName(context.Background(), uuid.New(), "")
	if err != domain.ErrInvalidInput {
		t.Errorf("expected ErrInvalidInput for empty name, got %v", err)
	}
}

func TestUserUseCase_GetFilter(t *testing.T) {
	uc, ur, _ := newUserUC()
	userID := uuid.New()
	raw := json.RawMessage(`{"my_email":"me@example.com","pull_conditions":{"to_me":true}}`)
	ur.users[userID] = &domain.User{ID: userID, FilterSettings: raw}

	f, err := uc.GetFilter(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	if f.MyEmail != "me@example.com" {
		t.Errorf("MyEmail = %q, want me@example.com", f.MyEmail)
	}
}

func TestUserUseCase_UpdateFilter(t *testing.T) {
	uc, ur, _ := newUserUC()
	userID := uuid.New()

	settings := &domain.FilterSettings{MyEmail: "me@example.com"}
	settings.PullConditions.ToMe = true

	if err := uc.UpdateFilter(context.Background(), userID, settings); err != nil {
		t.Fatal(err)
	}
	if ur.filterSettingSet == nil {
		t.Fatal("expected filter settings to be saved")
	}
	var parsed domain.FilterSettings
	if err := json.Unmarshal(ur.filterSettingSet, &parsed); err != nil {
		t.Fatalf("saved filter is not valid JSON: %v", err)
	}
	if parsed.MyEmail != "me@example.com" {
		t.Errorf("saved MyEmail = %q, want me@example.com", parsed.MyEmail)
	}
}

func TestUserUseCase_UpsertAPIKey_BYOK_Encrypts(t *testing.T) {
	uc, _, ar := newUserUC()
	userID := uuid.New()

	key := &domain.UserAPIKey{
		AIMode: domain.AIModeByok,
		APIKey: "sk-secret-key",
		Model:  "gpt-4o",
	}

	if err := uc.UpsertAPIKey(context.Background(), userID, key); err != nil {
		t.Fatal(err)
	}
	if ar.upsert == nil {
		t.Fatal("expected upsert to be called")
	}
	if ar.upsert.APIKey == "sk-secret-key" {
		t.Error("API key should be encrypted before storage")
	}
	if ar.upsert.APIKey != "enc:sk-secret-key" {
		t.Errorf("encrypted key = %q, want enc:sk-secret-key", ar.upsert.APIKey)
	}
	if ar.upsert.UserID != userID {
		t.Error("UserID should be set from parameter")
	}
}

func TestUserUseCase_UpsertAPIKey_PA_EncryptsWebhook(t *testing.T) {
	uc, _, ar := newUserUC()
	userID := uuid.New()

	key := &domain.UserAPIKey{
		AIMode:      domain.AIModePA,
		PAWebhookURL: "https://webhook.example.com/secret",
	}

	if err := uc.UpsertAPIKey(context.Background(), userID, key); err != nil {
		t.Fatal(err)
	}
	if ar.upsert.PAWebhookURL == "https://webhook.example.com/secret" {
		t.Error("webhook URL should be encrypted before storage")
	}
	if ar.upsert.PAWebhookURL != "enc:https://webhook.example.com/secret" {
		t.Errorf("encrypted URL = %q", ar.upsert.PAWebhookURL)
	}
}

func TestUserUseCase_DeleteAPIKey(t *testing.T) {
	uc, _, ar := newUserUC()
	userID := uuid.New()
	ar.keys[userID] = &domain.UserAPIKey{UserID: userID, AIMode: domain.AIModeByok}

	if err := uc.DeleteAPIKey(context.Background(), userID); err != nil {
		t.Fatal(err)
	}
	if _, ok := ar.keys[userID]; ok {
		t.Error("expected key to be deleted")
	}
}

func TestUserUseCase_GetAPIKey_MasksSensitiveFields(t *testing.T) {
	uc, _, ar := newUserUC()
	userID := uuid.New()
	ar.keys[userID] = &domain.UserAPIKey{
		UserID: userID,
		AIMode: domain.AIModeByok,
		APIKey: "enc:secret",
		Model:  "gpt-4o",
	}

	key, err := uc.GetAPIKey(context.Background(), userID)
	if err != nil {
		t.Fatal(err)
	}
	if key.APIKey != "" {
		t.Errorf("APIKey should be masked, got %q", key.APIKey)
	}
	if key.PAWebhookURL != "" {
		t.Errorf("PAWebhookURL should be masked, got %q", key.PAWebhookURL)
	}
	if key.Model != "gpt-4o" {
		t.Errorf("Model should be preserved, got %q", key.Model)
	}
}
