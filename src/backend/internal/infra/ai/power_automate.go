package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/buithean2010/mail-tracker/backend/internal/domain"
)

// PAClient posts the thread to a Power Automate webhook and parses Copilot JSON output (POC).
type PAClient struct {
	http *http.Client
}

func NewPAClient() *PAClient {
	return &PAClient{http: &http.Client{Timeout: httpTimeout}}
}

func (c *PAClient) Summarize(ctx context.Context, webhookURL string, messages []*domain.MailMessage) (*domain.SummaryResult, error) {
	body, _ := json.Marshal(map[string]any{
		"messages": messages,
		"prompt":   systemPrompt,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("PA webhook request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("PA webhook returned %d", resp.StatusCode)
	}

	// PA/Copilot may return the JSON directly or wrapped in a body field
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode PA response: %w", err)
	}

	// Try unwrapping a "body" envelope first, then fall back to top-level
	content, ok := raw["body"]
	if !ok {
		content, _ = json.Marshal(raw)
	}

	return parseAIResponse(string(content))
}
