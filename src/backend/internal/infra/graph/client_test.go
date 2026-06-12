package graph_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/buithean2010/mail-tracker/backend/internal/infra/graph"
)

// graphResp mirrors the Graph API envelope.
type graphResp struct {
	Value    []graphMsg `json:"value"`
	NextLink string     `json:"@odata.nextLink,omitempty"`
}

type graphMsg struct {
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

type msFrom struct{ EmailAddress msAddr `json:"emailAddress"` }
type msAddr struct {
	Address string `json:"address"`
	Name    string `json:"name"`
}
type msBody struct{ Content string `json:"content"` }

// roundTripFunc lets us use a plain function as an http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// jsonResp builds an http.Response with a JSON body.
func jsonResp(status int, body any) *http.Response {
	b, _ := json.Marshal(body)
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewReader(b)),
	}
}

// TestFetchMessages_SinglePage verifies correct parsing of a single-page response.
func TestFetchMessages_SinglePage(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	msg := graphMsg{
		ID:             "msg1",
		ConversationID: "conv1",
		Subject:        "Hello World",
		From:           msFrom{EmailAddress: msAddr{Address: "sender@example.com"}},
		ToRecipients:   []msAddr{{Address: "me@example.com"}},
		CcRecipients:   []msAddr{{Address: "cc@example.com"}},
		ReceivedAt:     now,
		BodyPreview:    "Preview",
		Body:           msBody{Content: "<p>Body</p>"},
		InternetMsgID:  "<msgid@mail>",
	}

	calls := 0
	rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Header.Get("Authorization") != "Bearer mytoken" {
			return jsonResp(http.StatusUnauthorized, nil), nil
		}
		return jsonResp(http.StatusOK, graphResp{Value: []graphMsg{msg}}), nil
	})

	client := graph.NewClientWithTransport("https://graph.microsoft.com/v1.0", rt)
	msgs, err := client.FetchMessages(context.Background(), "mytoken", time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 HTTP call, got %d", calls)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	m := msgs[0]
	if m.ID != "msg1" {
		t.Errorf("ID = %q, want msg1", m.ID)
	}
	if m.ConversationID != "conv1" {
		t.Errorf("ConversationID = %q, want conv1", m.ConversationID)
	}
	if m.Subject != "Hello World" {
		t.Errorf("Subject = %q, want Hello World", m.Subject)
	}
	if m.From != "sender@example.com" {
		t.Errorf("From = %q, want sender@example.com", m.From)
	}
	if !reflect.DeepEqual(m.ToRecipients, []string{"me@example.com"}) {
		t.Errorf("ToRecipients = %v", m.ToRecipients)
	}
	if !reflect.DeepEqual(m.CcRecipients, []string{"cc@example.com"}) {
		t.Errorf("CcRecipients = %v", m.CcRecipients)
	}
	if m.Body != "<p>Body</p>" {
		t.Errorf("Body = %q", m.Body)
	}
}

// TestFetchMessages_Pagination verifies that nextLink pages are followed.
func TestFetchMessages_Pagination(t *testing.T) {
	calls := 0
	rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		switch calls {
		case 1:
			// First page: return nextLink pointing to a second URL.
			return jsonResp(http.StatusOK, graphResp{
				Value:    []graphMsg{{ID: "msg1", ConversationID: "c1"}},
				NextLink: "https://graph.microsoft.com/v1.0/page2",
			}), nil
		case 2:
			// Second page: no nextLink.
			return jsonResp(http.StatusOK, graphResp{
				Value: []graphMsg{{ID: "msg2", ConversationID: "c2"}},
			}), nil
		default:
			return jsonResp(http.StatusInternalServerError, nil), nil
		}
	})

	client := graph.NewClientWithTransport("https://graph.microsoft.com/v1.0", rt)
	msgs, err := client.FetchMessages(context.Background(), "token", time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 HTTP calls, got %d", calls)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].ID != "msg1" || msgs[1].ID != "msg2" {
		t.Errorf("unexpected message IDs: %s, %s", msgs[0].ID, msgs[1].ID)
	}
}

// TestFetchMessages_EmptyResponse verifies an empty value array returns empty slice (not nil).
func TestFetchMessages_EmptyResponse(t *testing.T) {
	rt := roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return jsonResp(http.StatusOK, graphResp{Value: nil}), nil
	})

	client := graph.NewClientWithTransport("https://graph.microsoft.com/v1.0", rt)
	msgs, err := client.FetchMessages(context.Background(), "token", time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages, got %d", len(msgs))
	}
}

// TestFetchMessages_Unauthorized verifies a 401 response produces an error.
func TestFetchMessages_Unauthorized(t *testing.T) {
	rt := roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return jsonResp(http.StatusUnauthorized, nil), nil
	})

	client := graph.NewClientWithTransport("https://graph.microsoft.com/v1.0", rt)
	_, err := client.FetchMessages(context.Background(), "bad-token", time.Now().Add(-24*time.Hour))
	if err == nil {
		t.Error("expected error for 401 response, got nil")
	}
}

// TestFetchMessages_ServerError verifies a 500 response produces an error.
func TestFetchMessages_ServerError(t *testing.T) {
	rt := roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return jsonResp(http.StatusInternalServerError, nil), nil
	})

	client := graph.NewClientWithTransport("https://graph.microsoft.com/v1.0", rt)
	_, err := client.FetchMessages(context.Background(), "token", time.Now().Add(-24*time.Hour))
	if err == nil {
		t.Error("expected error for 500 response, got nil")
	}
}

// TestFetchMessages_BearerTokenForwarded verifies the access token is sent as Bearer.
func TestFetchMessages_BearerTokenForwarded(t *testing.T) {
	var gotAuth string
	rt := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		gotAuth = r.Header.Get("Authorization")
		return jsonResp(http.StatusOK, graphResp{}), nil
	})

	client := graph.NewClientWithTransport("https://graph.microsoft.com/v1.0", rt)
	_, _ = client.FetchMessages(context.Background(), "my-secret-token", time.Now().Add(-1*time.Hour))
	if gotAuth != "Bearer my-secret-token" {
		t.Errorf("Authorization header = %q, want Bearer my-secret-token", gotAuth)
	}
}
