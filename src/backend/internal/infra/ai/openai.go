package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/buithean2010/mail-tracker/backend/internal/domain"
)

const (
	openaiBaseURL = "https://api.openai.com/v1"
	httpTimeout   = 60 * time.Second
)

type OpenAIClient struct {
	baseURL string
	http    *http.Client
}

func NewOpenAIClient() *OpenAIClient {
	return &OpenAIClient{
		baseURL: openaiBaseURL,
		http:    &http.Client{Timeout: httpTimeout},
	}
}

func NewOpenRouterClient() *OpenAIClient {
	return &OpenAIClient{
		baseURL: "https://openrouter.ai/api/v1",
		http:    &http.Client{Timeout: httpTimeout},
	}
}

func (c *OpenAIClient) Summarize(ctx context.Context, apiKey, model string, messages []*domain.MailMessage) (*domain.SummaryResult, error) {
	if model == "" {
		model = "gpt-4o-mini"
	}

	prompt := buildPrompt(messages)

	reqBody, _ := json.Marshal(map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": prompt},
		},
		"temperature":     0.2,
		"response_format": map[string]string{"type": "json_object"},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai returned %d", resp.StatusCode)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode openai response: %w", err)
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("openai: no choices returned")
	}

	return parseAIResponse(result.Choices[0].Message.Content)
}

const systemPrompt = `You are an email analysis assistant. Analyze the email thread and respond with JSON only:
{
  "summary": "2-3 sentence summary of the thread",
  "priority": "high|medium|low",
  "action_required": true|false,
  "action_detail": "what action is needed, or empty string"
}`

func buildPrompt(messages []*domain.MailMessage) string {
	var sb strings.Builder
	sb.WriteString("Email thread:\n\n")
	for i, m := range messages {
		fmt.Fprintf(&sb, "--- Message %d ---\nFrom: %s\nDate: %s\nSubject: %s\n\n%s\n\n",
			i+1, m.From, m.ReceivedAt.Format("2006-01-02 15:04"), m.Subject, m.BodyPreview)
	}
	return sb.String()
}

func parseAIResponse(content string) (*domain.SummaryResult, error) {
	var r domain.SummaryResult
	if err := json.Unmarshal([]byte(content), &r); err != nil {
		return nil, fmt.Errorf("parse AI JSON: %w (content: %q)", err, content)
	}
	return &r, nil
}
