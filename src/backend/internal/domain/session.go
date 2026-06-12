package domain

import (
	"time"

	"github.com/google/uuid"
)

type Session struct {
	ID        string    `json:"id"` // random 32-byte hex
	UserID    uuid.UUID `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}
