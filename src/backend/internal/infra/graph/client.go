package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/buithean2010/mail-tracker/backend/internal/domain"
)

const (
	defaultBaseURL = "https://graph.microsoft.com/v1.0"
	selectFields   = "id,conversationId,subject,from,toRecipients,ccRecipients,receivedDateTime,bodyPreview,body,internetMessageId"
	timeout        = 30 * time.Second
)

type Client struct {
	http    *http.Client
	baseURL string
}

func NewClient() *Client {
	return &Client{http: &http.Client{Timeout: timeout}, baseURL: defaultBaseURL}
}

// NewClientWithBaseURL creates a client with a custom base URL (for testing).
func NewClientWithBaseURL(baseURL string) *Client {
	return &Client{http: &http.Client{Timeout: timeout}, baseURL: baseURL}
}

// NewClientWithTransport creates a client with a custom RoundTripper (for testing).
func NewClientWithTransport(baseURL string, rt http.RoundTripper) *Client {
	return &Client{http: &http.Client{Timeout: timeout, Transport: rt}, baseURL: baseURL}
}

// FetchMessages pulls all inbox messages received after `since`, following @odata.nextLink pages.
func (c *Client) FetchMessages(ctx context.Context, accessToken string, since time.Time) ([]*domain.MailMessage, error) {
	sinceStr := since.UTC().Format(time.RFC3339)
	q := url.Values{}
	q.Set("$select", selectFields)
	q.Set("$filter", "receivedDateTime ge "+sinceStr)
	q.Set("$orderby", "receivedDateTime desc")
	q.Set("$top", "100")
	endpoint := c.baseURL + "/me/mailFolders/Inbox/messages?" + q.Encode()

	var all []*domain.MailMessage
	maxPages := 50 // guard against runaway paging

	for i := 0; i < maxPages && endpoint != ""; i++ {
		msgs, next, err := c.fetchPage(ctx, accessToken, endpoint)
		if err != nil {
			return nil, err
		}
		all = append(all, msgs...)
		endpoint = next
	}

	return all, nil
}

type msMessageResponse struct {
	Value    []msMessage `json:"value"`
	NextLink string      `json:"@odata.nextLink"`
}

type msMessage struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversationId"`
	Subject        string    `json:"subject"`
	From           msFrom    `json:"from"`
	ToRecipients   []msAddr  `json:"toRecipients"`
	CcRecipients   []msAddr  `json:"ccRecipients"`
	ReceivedAt     time.Time `json:"receivedDateTime"`
	BodyPreview    string    `json:"bodyPreview"`
	Body           msBody    `json:"body"`
	InternetMsgID  string    `json:"internetMessageId"`
}

type msFrom struct {
	EmailAddress msAddr `json:"emailAddress"`
}

type msAddr struct {
	Address string `json:"address"`
	Name    string `json:"name"`
}

type msBody struct {
	Content string `json:"content"`
}

func (c *Client) fetchPage(ctx context.Context, token, endpoint string) ([]*domain.MailMessage, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("graph request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, "", fmt.Errorf("graph 401: token may be expired")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("graph returned %d", resp.StatusCode)
	}

	var result msMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, "", fmt.Errorf("decode graph response: %w", err)
	}

	msgs := make([]*domain.MailMessage, 0, len(result.Value))
	for _, m := range result.Value {
		to := make([]string, len(m.ToRecipients))
		for i, a := range m.ToRecipients {
			to[i] = a.Address
		}
		cc := make([]string, len(m.CcRecipients))
		for i, a := range m.CcRecipients {
			cc[i] = a.Address
		}

		msgs = append(msgs, &domain.MailMessage{
			ID:             m.ID,
			ConversationID: m.ConversationID,
			Subject:        m.Subject,
			From:           m.From.EmailAddress.Address,
			ToRecipients:   to,
			CcRecipients:   cc,
			ReceivedAt:     m.ReceivedAt,
			BodyPreview:    m.BodyPreview,
			Body:           m.Body.Content,
			InternetMsgID:  m.InternetMsgID,
		})
	}

	return msgs, result.NextLink, nil
}
