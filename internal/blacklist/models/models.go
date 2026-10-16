// Package models holds the data structures backing the online blacklist feature.
package models

import "time"

const (
	// ActionAdd marks a "contact added to the online blacklist" log entry.
	ActionAdd = "add"
	// ActionRemove marks a "contact removed from the online blacklist" log entry.
	ActionRemove = "remove"
)

// Blacklist is a row of `contact_blacklist`. A contact is considered blocked
// for as long as a row exists for them.
type Blacklist struct {
	ID        int       `db:"id" json:"id"`
	ContactID int       `db:"contact_id" json:"contact_id"`
	Reason    string    `db:"reason" json:"reason"`
	CreatedBy int       `db:"created_by" json:"created_by"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// Status is the blacklist state of a contact, returned to the UI so it can
// toggle the add/remove action.
type Status struct {
	Blocked   bool       `json:"blocked"`
	Reason    string     `json:"reason"`
	CreatedBy int        `json:"created_by"`
	CreatedAt *time.Time `json:"created_at"`
}

// BlacklistLog is an append-only entry of every add/remove operation.
type BlacklistLog struct {
	ID        int       `db:"id" json:"id"`
	ContactID int       `db:"contact_id" json:"contact_id"`
	Action    string    `db:"action" json:"action"`
	Reason    string    `db:"reason" json:"reason"`
	ActorID   int       `db:"actor_id" json:"actor_id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// Customer is one row of the "客户信息" (customer) list. The actor/reason/time
// columns are only populated for contacts that are currently blacklisted.
type Customer struct {
	Total                  int        `db:"total" json:"-"`
	ContactID              int        `db:"contact_id" json:"contact_id"`
	ContactName            string     `db:"contact_name" json:"contact_name"`
	ContactType            string     `db:"contact_type" json:"contact_type"`
	ContactCreatedAt       time.Time  `db:"contact_created_at" json:"contact_created_at"`
	ExternalUserID         string     `db:"external_user_id" json:"external_user_id"`
	PhoneNumber            string     `db:"phone_number" json:"phone_number"`
	PhoneNumberCountryCode string     `db:"phone_number_country_code" json:"phone_number_country_code"`
	ContactEmail           string     `db:"contact_email" json:"contact_email"`
	Blocked                bool       `db:"blocked" json:"blocked"`
	Reason                 string     `db:"reason" json:"reason"`
	BlacklistedAt          *time.Time `db:"blacklisted_at" json:"blacklisted_at"`
	ActorID                int        `db:"actor_id" json:"actor_id"`
	ActorName              string     `db:"actor_name" json:"actor_name"`
}

// Log is one row of the "操作日志" (operation log) list.
type Log struct {
	Total       int       `db:"total" json:"-"`
	ID          int       `db:"id" json:"id"`
	ContactID   int       `db:"contact_id" json:"contact_id"`
	ContactName string    `db:"contact_name" json:"contact_name"`
	Action      string    `db:"action" json:"action"`
	Reason      string    `db:"reason" json:"reason"`
	ActorID     int       `db:"actor_id" json:"actor_id"`
	ActorName   string    `db:"actor_name" json:"actor_name"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}
