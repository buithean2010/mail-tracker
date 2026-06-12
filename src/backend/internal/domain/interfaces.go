package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type UserRepo interface {
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByMicrosoftID(ctx context.Context, msID string) (*User, error)
	Upsert(ctx context.Context, user *User) error
	UpdateFilterSettings(ctx context.Context, userID uuid.UUID, settings json.RawMessage) error
	UpdateDisplayName(ctx context.Context, userID uuid.UUID, name string) error
	SetNeedsReauth(ctx context.Context, userID uuid.UUID, flag bool) error
}

type SessionRepo interface {
	Create(ctx context.Context, session *Session) error
	GetByID(ctx context.Context, id string) (*Session, error)
	Delete(ctx context.Context, id string) error
	DeleteExpired(ctx context.Context) error
	GetActiveUsers(ctx context.Context, withinDays int) ([]*User, error)
}

type TokenRepo interface {
	Upsert(ctx context.Context, token *UserToken) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (*UserToken, error)
	Delete(ctx context.Context, userID uuid.UUID) error
}

type ThreadRepo interface {
	Upsert(ctx context.Context, thread *EmailThread) error
	ListByUser(ctx context.Context, userID uuid.UUID, filter ThreadFilter) ([]*EmailThread, int, error)
	GetByID(ctx context.Context, id uuid.UUID) (*EmailThread, error)
	Update(ctx context.Context, thread *EmailThread) error
	GetPendingSummaries(ctx context.Context) ([]*EmailThread, error)
	UpdateSummaryStatus(ctx context.Context, id uuid.UUID, status string) error
	UpdateSummary(ctx context.Context, id uuid.UUID, result *SummaryResult) error
}

type APIKeyRepo interface {
	Upsert(ctx context.Context, key *UserAPIKey) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (*UserAPIKey, error)
	Delete(ctx context.Context, userID uuid.UUID) error
}

type MailClient interface {
	FetchMessages(ctx context.Context, accessToken string, since time.Time) ([]*MailMessage, error)
}

type AIClient interface {
	Summarize(ctx context.Context, apiKey, model string, messages []*MailMessage) (*SummaryResult, error)
}

type PAClient interface {
	Summarize(ctx context.Context, webhookURL string, messages []*MailMessage) (*SummaryResult, error)
}

type Encryptor interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

type SSEBroker interface {
	Publish(userID uuid.UUID, event string, data any)
	Subscribe(userID uuid.UUID) (<-chan SSEEvent, func())
}

type SSEEvent struct {
	Event string
	Data  string
}
