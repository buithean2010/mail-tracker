package domain

import (
	"time"

	"github.com/google/uuid"
)

type UserToken struct {
	UserID       uuid.UUID `json:"user_id"`
	AccessToken  string    `json:"-"` // plaintext in memory, encrypted at rest
	RefreshToken string    `json:"-"`
	ExpiresAt    time.Time `json:"expires_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (t *UserToken) NeedsRefresh() bool {
	return time.Until(t.ExpiresAt) < 5*time.Minute
}
