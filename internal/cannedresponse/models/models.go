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
