package models

import (
	"time"

	"github.com/lib/pq"
)

// InboxQuickReplyConfig holds the automatic reply configuration for an inbox.
type InboxQuickReplyConfig struct {
	ID                  int64     `db:"id" json:"id"`
	InboxID             int64     `db:"inbox_id" json:"inbox_id"`
	WelcomeMessage      string    `db:"welcome_message" json:"welcome_message"`
	TransferKeyword     string    `db:"transfer_keyword" json:"transfer_keyword"`
	QueueReply          string    `db:"queue_reply" json:"queue_reply"`
	AssignedReply       string    `db:"assigned_reply" json:"assigned_reply"`
	ClosedReply         string    `db:"closed_reply" json:"closed_reply"`
	NoReplyTimeoutReply string    `db:"no_reply_timeout_reply" json:"no_reply_timeout_reply"`
	Enabled             bool      `db:"enabled" json:"enabled"`
	CreatedAt           time.Time `db:"created_at" json:"created_at"`
	UpdatedAt           time.Time `db:"updated_at" json:"updated_at"`
}

// QuickReplyTopic is a topic (category) under which quick reply questions
// are grouped. Each topic belongs to an inbox.
// Names stores all display names (aliases) for this topic. The first element
// is the primary name; remaining elements are aliases. All names map to the
// same set of questions.
type QuickReplyTopic struct {
	ID          int64                `db:"id" json:"id"`
	InboxID     int64                `db:"inbox_id" json:"inbox_id"`
	Name        string               `db:"name" json:"name"`
	Names       pq.StringArray       `db:"names" json:"names"`
	HintMessage string               `db:"hint_message" json:"hint_message"`
	SortOrder   int                  `db:"sort_order" json:"sort_order"`
	Questions   []QuickReplyQuestion `db:"-" json:"questions"`
}

// QuickReplyQuestion is a question under a topic that carries an automatic answer.
type QuickReplyQuestion struct {
	ID        int64  `db:"id" json:"id"`
	TopicID   int64  `db:"topic_id" json:"topic_id"`
	Question  string `db:"question" json:"question"`
	Answer    string `db:"answer" json:"answer"`
	SortOrder int    `db:"sort_order" json:"sort_order"`
}
