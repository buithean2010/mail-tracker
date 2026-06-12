package domain_test

import (
	"encoding/json"
	"testing"

	"github.com/buithean2010/mail-tracker/backend/internal/domain"
)

func TestParseFilterSettings_Empty(t *testing.T) {
	for _, raw := range []json.RawMessage{nil, json.RawMessage("null"), json.RawMessage("{}")} {
		f, err := domain.ParseFilterSettings(raw)
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", raw, err)
		}
		if f == nil {
			t.Fatal("expected non-nil FilterSettings")
		}
	}
}

func TestParseFilterSettings_Valid(t *testing.T) {
	raw := json.RawMessage(`{
		"my_email": "me@example.com",
		"my_name": "Alice",
		"my_name_aliases": ["Al"],
		"pull_conditions": {"to_me": true, "cc_me": false},
		"exclude": {"senders": ["spam@example.com"], "subject_keywords": ["newsletter"]}
	}`)
	f, err := domain.ParseFilterSettings(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.MyEmail != "me@example.com" {
		t.Errorf("MyEmail = %q, want me@example.com", f.MyEmail)
	}
	if f.MyName != "Alice" {
		t.Errorf("MyName = %q, want Alice", f.MyName)
	}
	if len(f.MyNameAliases) != 1 || f.MyNameAliases[0] != "Al" {
		t.Errorf("MyNameAliases = %v, want [Al]", f.MyNameAliases)
	}
	if !f.PullConditions.ToMe {
		t.Error("expected PullConditions.ToMe = true")
	}
	if len(f.Exclude.Senders) != 1 || f.Exclude.Senders[0] != "spam@example.com" {
		t.Errorf("Exclude.Senders = %v", f.Exclude.Senders)
	}
}

func TestParseFilterSettings_Invalid(t *testing.T) {
	_, err := domain.ParseFilterSettings(json.RawMessage(`{invalid json}`))
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func newFilter() *domain.FilterSettings {
	f := &domain.FilterSettings{
		MyEmail: "me@example.com",
		MyName:  "Alice",
	}
	f.MyNameAliases = []string{"Al", "Ally"}
	f.PullConditions.ToMe = true
	f.PullConditions.CcMe = true
	f.PullConditions.MentionEmail = true
	f.PullConditions.MentionName = true
	f.PullConditions.MentionAliases = true
	f.Exclude.SubjectKeywords = []string{"newsletter", "unsubscribe"}
	f.Exclude.Senders = []string{"noreply@spam.com"}
	return f
}

func TestFilterSettingsMatches(t *testing.T) {
	f := newFilter()

	tests := []struct {
		name string
		msg  *domain.MailMessage
		want bool
	}{
		{
			name: "to_me matches",
			msg:  &domain.MailMessage{Subject: "Hello", From: "boss@work.com", ToRecipients: []string{"me@example.com"}},
			want: true,
		},
		{
			name: "to_me case-insensitive",
			msg:  &domain.MailMessage{Subject: "Hello", From: "boss@work.com", ToRecipients: []string{"ME@EXAMPLE.COM"}},
			want: true,
		},
		{
			name: "cc_me matches",
			msg:  &domain.MailMessage{Subject: "FYI", From: "boss@work.com", CcRecipients: []string{"me@example.com"}},
			want: true,
		},
		{
			name: "mention_email in body",
			msg:  &domain.MailMessage{Subject: "Hello", From: "boss@work.com", Body: "Please contact me@example.com ASAP"},
			want: true,
		},
		{
			name: "mention_name in body",
			msg:  &domain.MailMessage{Subject: "Hello", From: "boss@work.com", Body: "Hi alice, please review"},
			want: true,
		},
		{
			name: "mention_alias in body",
			msg:  &domain.MailMessage{Subject: "Hello", From: "boss@work.com", Body: "Hey Ally can you help"},
			want: true,
		},
		{
			name: "excluded subject keyword",
			msg:  &domain.MailMessage{Subject: "Monthly Newsletter", From: "boss@work.com", ToRecipients: []string{"me@example.com"}},
			want: false,
		},
		{
			name: "excluded subject keyword case-insensitive",
			msg:  &domain.MailMessage{Subject: "NEWSLETTER update", From: "boss@work.com", ToRecipients: []string{"me@example.com"}},
			want: false,
		},
		{
			name: "excluded sender",
			msg:  &domain.MailMessage{Subject: "Hello", From: "noreply@spam.com", ToRecipients: []string{"me@example.com"}},
			want: false,
		},
		{
			name: "no pull condition met",
			msg:  &domain.MailMessage{Subject: "Random email", From: "someone@work.com", Body: "Nothing relevant"},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := f.Matches(tc.msg)
			if got != tc.want {
				t.Errorf("Matches() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFilterSettingsMatches_ExcludeBeforeInclude(t *testing.T) {
	f := newFilter()
	// Message would match to_me but is excluded by sender — exclude wins.
	msg := &domain.MailMessage{
		Subject:      "Hello",
		From:         "noreply@spam.com",
		ToRecipients: []string{"me@example.com"},
	}
	if f.Matches(msg) {
		t.Error("excluded sender should prevent match even when to_me is set")
	}
}

func TestFilterSettingsMatches_NoConditionsConfigured(t *testing.T) {
	f := &domain.FilterSettings{MyEmail: "me@example.com"}
	msg := &domain.MailMessage{Subject: "Hello", From: "boss@work.com", ToRecipients: []string{"me@example.com"}}
	if f.Matches(msg) {
		t.Error("with no pull_conditions enabled, Matches should return false")
	}
}
