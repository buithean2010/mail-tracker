package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID             uuid.UUID       `json:"id"`
	MicrosoftID    string          `json:"microsoft_id"`
	Email          string          `json:"email"`
	DisplayName    string          `json:"display_name"`
	FilterSettings json.RawMessage `json:"filter_settings"`
	NeedsReauth    bool            `json:"needs_reauth"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}
