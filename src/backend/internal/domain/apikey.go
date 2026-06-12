package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	AIModeByok          = "byok"
	AIModePA            = "power_automate"
	AIProviderOpenAI    = "openai"
	AIProviderOpenRouter = "openrouter"
)

type UserAPIKey struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	AIMode       string    `json:"ai_mode"`
	Provider     string    `json:"provider"`
	APIKey       string    `json:"-"` // plaintext in memory, encrypted at rest
	Model        string    `json:"model"`
	PAWebhookURL string    `json:"-"` // encrypted at rest
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
