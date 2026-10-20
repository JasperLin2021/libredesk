// Package models holds the data structures of the keyword match (关键字匹配)
// feature: global rules that are matched against visitor / agent messages and
// trigger an automatic reply or an agent-only system alert.
package models

import "time"

const (
	// Message sources a rule can be applied to.
	SourceVisitor = "visitor"
	SourceAgent   = "agent"

	// Supported match types.
	MatchTypeKeyword = "keyword"
	MatchTypeRegex   = "regex"

	// Actions executed when a rule matches.
	// ActionAgentNotify only alerts agents (nothing is written to the
	// conversation), ActionAgentAndVisitor sends the reply into the
	// conversation so both the agent and the visitor see it.
	ActionAgentNotify     = "agent_notify"
	ActionAgentAndVisitor = "agent_and_visitor"

	// MatchPlaceholder is replaced with the text that actually matched the
	// rule inside the reply content.
	MatchPlaceholder = "##"
)

// KeywordMatchRule is a single global keyword match rule.
type KeywordMatchRule struct {
	ID            int       `db:"id" json:"id"`
	Name          string    `db:"name" json:"name"`
	Source        string    `db:"source" json:"source"`
	MatchType     string    `db:"match_type" json:"match_type"`
	Pattern       string    `db:"pattern" json:"pattern"`
	CaseSensitive bool      `db:"case_sensitive" json:"case_sensitive"`
	SearchFirst   bool      `db:"search_first" json:"search_first"`
	Action        string    `db:"action" json:"action"`
	ReplyContent  string    `db:"reply_content" json:"reply_content"`
	Enabled       bool      `db:"enabled" json:"enabled"`
	SortOrder     int       `db:"sort_order" json:"sort_order"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at" json:"updated_at"`
}

// Match is the result of a rule matching a message: the rule, the text that
// matched it and the reply content with the `##` placeholder already replaced.
type Match struct {
	Rule         KeywordMatchRule `json:"rule"`
	MatchedText  string           `json:"matched_text"`
	ReplyContent string           `json:"reply_content"`
}

// IsValidSource reports whether the given source is supported.
func IsValidSource(source string) bool {
	return source == SourceVisitor || source == SourceAgent
}

// IsValidMatchType reports whether the given match type is supported.
func IsValidMatchType(matchType string) bool {
	return matchType == MatchTypeKeyword || matchType == MatchTypeRegex
}

// IsValidAction reports whether the given action is supported.
func IsValidAction(action string) bool {
	return action == ActionAgentNotify || action == ActionAgentAndVisitor
}
