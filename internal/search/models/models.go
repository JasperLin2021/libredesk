package models

import (
	"time"

	"github.com/volatiletech/null/v9"
)

// Filters is the optional filter set applied to a search query.
// The zero value is treated as "no filter" so callers that do not need
// filtering can simply pass `models.Filters{}` and keep the legacy behaviour.
type Filters struct {
	// StartAt and EndAt define the inclusive time window applied to the
	// conversation's created_at column. A zero time.Time means the bound
	// is not applied.
	StartAt time.Time
	EndAt   time.Time
	// InboxIDs restricts results to conversations whose inbox is in this
	// list. nil or an empty slice means no restriction.
	InboxIDs []int
	// AgentID restricts results to conversations assigned to this user.
	// Zero means no restriction.
	AgentID int
	// Sort controls the time ordering of results. Allowed values are
	// "newest" (default, latest first) and "oldest" (earliest first).
	// Any other value falls back to "newest".
	Sort string
}

type ConversationResult struct {
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UUID            string    `db:"uuid" json:"uuid"`
	ReferenceNumber string    `db:"reference_number" json:"reference_number"`
	Subject         string    `db:"subject" json:"subject"`
	Status          string    `db:"status" json:"status"`
}

type MessageResult struct {
	UUID                        string    `db:"message_uuid" json:"message_uuid"`
	CreatedAt                   time.Time `db:"created_at" json:"created_at"`
	TextContent                 string    `db:"text_content" json:"text_content"`
	ConversationCreatedAt       time.Time `db:"conversation_created_at" json:"conversation_created_at"`
	ConversationUUID            string    `db:"conversation_uuid" json:"conversation_uuid"`
	ConversationReferenceNumber string    `db:"conversation_reference_number" json:"conversation_reference_number"`
	ConversationStatus          string    `db:"conversation_status" json:"conversation_status"`
}

type ContactResult struct {
	ID             int         `db:"id" json:"id"`
	CreatedAt      time.Time   `db:"created_at" json:"created_at"`
	FirstName      string      `db:"first_name" json:"first_name"`
	LastName       string      `db:"last_name" json:"last_name"`
	Email          string      `db:"email" json:"email"`
	ExternalUserID null.String `db:"external_user_id" json:"external_user_id"`
}
