package domain

import (
	"encoding/json"
	"strings"
)

type FilterSettings struct {
	MyEmail       string   `json:"my_email"`
	MyName        string   `json:"my_name"`
	MyNameAliases []string `json:"my_name_aliases"`
	PullConditions struct {
		ToMe           bool `json:"to_me"`
		CcMe           bool `json:"cc_me"`
		MentionEmail   bool `json:"mention_email"`
		MentionName    bool `json:"mention_name"`
		MentionAliases bool `json:"mention_aliases"`
	} `json:"pull_conditions"`
	Exclude struct {
		Senders         []string `json:"senders"`
		SubjectKeywords []string `json:"subject_keywords"`
	} `json:"exclude"`
}

func ParseFilterSettings(raw json.RawMessage) (*FilterSettings, error) {
	if len(raw) == 0 || string(raw) == "null" || string(raw) == "{}" {
		return &FilterSettings{}, nil
	}
	var f FilterSettings
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, err
	}
	return &f, nil
}

func (f *FilterSettings) Matches(msg *MailMessage) bool {
	subjectLower := strings.ToLower(msg.Subject)
	for _, kw := range f.Exclude.SubjectKeywords {
		if strings.Contains(subjectLower, strings.ToLower(kw)) {
			return false
		}
	}

	fromLower := strings.ToLower(msg.From)
	for _, s := range f.Exclude.Senders {
		if strings.Contains(fromLower, strings.ToLower(s)) {
			return false
		}
	}

	if f.PullConditions.ToMe && containsAddr(msg.ToRecipients, f.MyEmail) {
		return true
	}
	if f.PullConditions.CcMe && containsAddr(msg.CcRecipients, f.MyEmail) {
		return true
	}
	if f.PullConditions.MentionEmail {
		if strings.Contains(strings.ToLower(msg.Body), strings.ToLower(f.MyEmail)) {
			return true
		}
	}
	if f.PullConditions.MentionName && f.MyName != "" {
		if strings.Contains(strings.ToLower(msg.Body), strings.ToLower(f.MyName)) {
			return true
		}
	}
	if f.PullConditions.MentionAliases {
		for _, alias := range f.MyNameAliases {
			if strings.Contains(strings.ToLower(msg.Body), strings.ToLower(alias)) {
				return true
			}
		}
	}

	return false
}

func containsAddr(list []string, addr string) bool {
	addr = strings.ToLower(addr)
	for _, a := range list {
		if strings.ToLower(a) == addr {
			return true
		}
	}
	return false
}
