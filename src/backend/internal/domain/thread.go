package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	SummaryStatusPending    = "pending"
	SummaryStatusProcessing = "processing"
	SummaryStatusDone       = "done"
	SummaryStatusFailed     = "failed"
	SummaryStatusNoKey      = "no_key"
)

type EmailThread struct {
	ID             uuid.UUID       `json:"id"`
	UserID         uuid.UUID       `json:"user_id"`
	ConversationID string          `json:"conversation_id"`
	Subject        string          `json:"subject"`
	FromAddress    string          `json:"from"`
	ReceivedAt     time.Time       `json:"received_at"`
	LastMessageAt  time.Time       `json:"last_message_at"`
	MessageCount   int             `json:"message_count"`
	RawMessages    json.RawMessage `json:"raw_messages"`

	// Tier 2 fields
	Summary        string `json:"summary"`
	Priority       string `json:"priority"`
	ActionRequired bool   `json:"action_required"`
	ActionDetail   string `json:"action_detail"`
	Status         string `json:"status"`
	Notes          string `json:"notes"`
	SummaryStatus  string `json:"summary_status"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ThreadFilter struct {
	Status   string
	Priority string
	From     string
	Search   string
	DateFrom *time.Time
	DateTo   *time.Time
	Page     int
	Limit    int
}

type MailMessage struct {
	ID              string          `json:"id"`
	ConversationID  string          `json:"conversationId"`
	Subject         string          `json:"subject"`
	From            string          `json:"from"`
	ToRecipients    []string        `json:"toRecipients"`
	CcRecipients    []string        `json:"ccRecipients"`
	ReceivedAt      time.Time       `json:"receivedDateTime"`
	BodyPreview     string          `json:"bodyPreview"`
	Body            string          `json:"body"`
	InternetMsgID   string          `json:"internetMessageId"`
}

type SummaryResult struct {
	Summary        string `json:"summary"`
	Priority       string `json:"priority"`
	ActionRequired bool   `json:"action_required"`
	ActionDetail   string `json:"action_detail"`
}
