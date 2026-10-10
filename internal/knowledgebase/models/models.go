package models

import "time"

// ArticleStatus distinguishes published knowledge from draft knowledge.
// Drafts are surfaced in the UI under a virtual "草稿" node filtered by
// `status='draft'`.
type ArticleStatus string

const (
	ArticleStatusPublished ArticleStatus = "published"
	ArticleStatusDraft     ArticleStatus = "draft"
)

// Category is a two-level category under which knowledge articles are
// grouped. Level-1 rows (ParentID == nil) are the "知识库" groups; level-2
// rows belong to a level-1 parent. The same row can also be uniquely
// identified by an optional Code so imported knowledge can reference
// categories by their spreadsheet code.
type Category struct {
	ID        int64     `db:"id" json:"id"`
	ParentID  *int64    `db:"parent_id" json:"parent_id"`
	Code      *string   `db:"code" json:"code,omitempty"`
	Name      string    `db:"name" json:"name"`
	SortOrder int       `db:"sort_order" json:"sort_order"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`

	// Children holds the nested sub-categories of this category.
	Children []*Category `db:"-" json:"children"`
	// ArticleCount is the number of articles directly under this category.
	// It is materialized by the listing query so the UI can show counts
	// without an extra round-trip.
	ArticleCount int `db:"article_count" json:"article_count"`
}

// Article is a single knowledge article belonging to a category.
type Article struct {
	ID         int64        `db:"id" json:"id"`
	CategoryID *int64       `db:"category_id" json:"category_id"`
	ExternalID *string      `db:"external_id" json:"external_id,omitempty"`
	Code       *string      `db:"code" json:"code,omitempty"`
	Title      string       `db:"title" json:"title"`
	Author     *string      `db:"author" json:"author,omitempty"`
	Content    string       `db:"content" json:"content"`
	Status     ArticleStatus `db:"status" json:"status"`
	SortOrder  int          `db:"sort_order" json:"sort_order"`
	CreatedAt  time.Time    `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time    `db:"updated_at" json:"updated_at"`

	// Optional joined fields populated by list / detail queries.
	CategoryName     *string `db:"category_name" json:"category_name,omitempty"`
	ParentCategoryID *int64  `db:"parent_category_id" json:"parent_category_id,omitempty"`
	ParentCategoryName *string `db:"parent_category_name" json:"parent_category_name,omitempty"`
	Summary          *string `db:"summary" json:"summary,omitempty"`

	// Attachments are populated by the detail query.
	Attachments []Attachment `db:"-" json:"attachments,omitempty"`
}

// Attachment links an article to a media record stored in the existing
// `media` table. Uploads are sent through `POST /api/v1/media` and
// referenced by their UUID.
type Attachment struct {
	ID          int64     `db:"id" json:"id"`
	ArticleID   int64     `db:"article_id" json:"article_id"`
	MediaUUID   string    `db:"media_uuid" json:"media_uuid"`
	FileName    *string   `db:"file_name" json:"file_name,omitempty"`
	ContentType *string   `db:"content_type" json:"content_type,omitempty"`
	Size        *int64    `db:"size" json:"size,omitempty"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

// AttachmentInput is the payload clients send when creating or updating
// articles. The media file is uploaded separately through the media
// endpoint; only its UUID and metadata are persisted here.
type AttachmentInput struct {
	MediaUUID   string  `json:"media_uuid"`
	FileName    *string `json:"file_name,omitempty"`
	ContentType *string `json:"content_type,omitempty"`
	Size        *int64  `json:"size,omitempty"`
}

// ArticleListResult is the paginated response returned by List.
type ArticleListResult struct {
	Total    int       `json:"total"`
	Page     int       `json:"page"`
	PerPage  int       `json:"per_page"`
	Articles []Article `json:"articles"`
}
