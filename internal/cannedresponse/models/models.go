package models

import "time"

// Category is a (possibly nested) category under which canned responses are
// grouped. Categories are global (not scoped to an inbox) and support
// arbitrary nesting via ParentID. A nil ParentID indicates a top-level
// category.
type Category struct {
	ID        int64     `db:"id" json:"id"`
	ParentID  *int64    `db:"parent_id" json:"parent_id"`
	Name      string    `db:"name" json:"name"`
	SortOrder int       `db:"sort_order" json:"sort_order"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`

	// Children holds the nested sub-categories of this category.
	Children []*Category `db:"-" json:"children"`
	// Responses holds the canned responses that belong directly to this
	// category (not to its children).
	Responses []Response `db:"-" json:"responses"`
}

// Response is a single canned response belonging to a category.
type Response struct {
	ID         int64     `db:"id" json:"id"`
	CategoryID int64     `db:"category_id" json:"category_id"`
	Content    string    `db:"content" json:"content"`
	SortOrder  int       `db:"sort_order" json:"sort_order"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
}

// PinSource distinguishes the two kinds of canned responses an agent can pin
// in the sidebar's "置顶常用语" group.
type PinSource string

const (
	// PinSourceGlobal references a row in the global `canned_responses` table.
	PinSourceGlobal PinSource = "global"
	// PinSourcePersonal references a row in the per-user
	// `user_canned_responses` table.
	PinSourcePersonal PinSource = "personal"
)

// PersonalResponse is a canned response that belongs to a single agent and
// appears under the "个人常用语" group in the sidebar.
type PersonalResponse struct {
	ID        int64     `db:"id" json:"id"`
	UserID    int64     `db:"user_id" json:"user_id"`
	Content   string    `db:"content" json:"content"`
	SortOrder int       `db:"sort_order" json:"sort_order"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// PinnedResponse is one entry in an agent's "置顶常用语" list. It is
// materialized at read time by joining a `user_canned_pins` row with the
// content of the underlying canned response (either global or personal), so
// the frontend never has to do another lookup to render the pin.
type PinnedResponse struct {
	// Source mirrors the PinSource of the row in user_canned_pins.
	Source PinSource `json:"source"`
	// ResponseID is the id of the underlying response in its source table.
	ResponseID int64 `json:"response_id"`
	// Content is the resolved canned response text at the time the view was
	// built. Admin edits to a global response are reflected on the next
	// reload.
	Content string `json:"content"`
	// SortOrder preserves the ordering chosen by the user for their pins.
	SortOrder int `json:"sort_order"`
}

// AgentView is the aggregate payload returned to the agent's sidebar. It
// combines the global canned-response category tree with the agent's personal
// responses and pin list in a single round-trip.
type AgentView struct {
	Pinned  []PinnedResponse `json:"pinned"`
	Personal []PersonalResponse `json:"personal"`
	Global  []*Category      `json:"global"`
}
