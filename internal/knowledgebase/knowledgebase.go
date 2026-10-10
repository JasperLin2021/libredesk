// Package knowledgebase implements the management of the "客服知识库" (knowledge
// base) feature: a two-level category tree of articles with rich-text bodies,
// rich filtering, bulk import/export, and per-article attachments stored as
// references to the shared media table.
package knowledgebase

import (
	"database/sql"
	"embed"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/knowledgebase/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/lib/pq"
	"github.com/zerodha/logf"
)

// Length limits mirror the constraints enforced by the download template and
// the database CHECK constraints.
const (
	// MaxCategoryNameLength is the maximum number of characters allowed in
	// a knowledge base category name.
	MaxCategoryNameLength = 50
	// MaxTitleLength is the maximum number of characters allowed in an
	// article title. The DB schema uses 200 to leave room for future
	// bulk-import notes, but the UI template advertises 50 as the soft
	// limit (titles longer than 50 fail to import per the spec).
	MaxTitleLength = 200
)

// Keyword scope values accepted by List and Import.
const (
	KeywordScopeAll     = "all"
	KeywordScopeTitle   = "title"
	KeywordScopeContent = "content"
)

// Sort orders accepted by List.
const (
	SortDesc = "desc"
	SortAsc  = "asc"
)

// Date fields the DateFrom/DateTo window can be applied to.
const (
	DateFieldCreated = "created_at"
	DateFieldUpdated = "updated_at"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

// Manager manages knowledge base categories, articles and imports.
type Manager struct {
	q    queries
	db   *sqlx.DB
	lo   *logf.Logger
	i18n *i18n.I18n
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB   *sqlx.DB
	Lo   *logf.Logger
	I18n *i18n.I18n
}

// queries contains prepared SQL queries. The struct field tag `query:"name"`
// is used by goyesqlx to bind prepared statements by the `-- name:` header
// in queries.sql.
type queries struct {
	GetAllCategories              *sqlx.Stmt `query:"get-all-categories"`
	GetCategory                   *sqlx.Stmt `query:"get-category"`
	GetCategoryByCode             *sqlx.Stmt `query:"get-category-by-code"`
	GetCategoryByParentAndName    *sqlx.Stmt `query:"get-category-by-parent-and-name"`
	InsertCategory                *sqlx.Stmt `query:"insert-category"`
	InsertCategoryOnConflict      *sqlx.Stmt `query:"insert-category-on-conflict"`
	UpdateCategory                *sqlx.Stmt `query:"update-category"`
	DeleteCategory                *sqlx.Stmt `query:"delete-category"`
	NextCategorySortOrder         *sqlx.Stmt `query:"next-category-sort-order"`
	CountArticlesByCategory       *sqlx.Stmt `query:"count-articles-by-category"`

	ArticleExistsInOtherCategories *sqlx.Stmt `query:"article-exists-in-other-categories"`
	GetArticleByExternalID         *sqlx.Stmt `query:"get-article-by-external-id"`

	GetArticle           *sqlx.Stmt `query:"get-article"`
	GetArticleAttachments *sqlx.Stmt `query:"get-article-attachments"`
	ListArticles         *sqlx.Stmt `query:"list-articles"`
	CountArticles        *sqlx.Stmt `query:"count-articles"`
	InsertArticle        *sqlx.Stmt `query:"insert-article"`
	InsertArticleOnConflict *sqlx.Stmt `query:"insert-article-on-conflict"`
	UpdateArticle        *sqlx.Stmt `query:"update-article"`
	DeleteArticle        *sqlx.Stmt `query:"delete-article"`

	InsertArticleAttachment  *sqlx.Stmt `query:"insert-article-attachment"`
	DeleteArticleAttachments *sqlx.Stmt `query:"delete-article-attachments"`

	ListAllArticles *sqlx.Stmt `query:"list-all-articles"`
}

// New creates and returns a new instance of the Manager.
func New(opts Opts) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{
		q:    q,
		db:   opts.DB,
		lo:   opts.Lo,
		i18n: opts.I18n,
	}, nil
}

// ListFilter captures the optional filter / pagination parameters accepted by
// List. Zero values select the entire knowledge base ordered by newest first.
type ListFilter struct {
	CategoryID         *int64
	ParentCategoryID   *int64
	Status             *models.ArticleStatus
	DateFrom           *time.Time
	DateTo             *time.Time
	// DateField selects which timestamp column the DateFrom/DateTo window
	// applies to: "created_at" (default) or "updated_at".
	DateField          string
	KeywordScope       string
	Keywords           []string
	SortOrder          string
	Page               int
	PerPage            int
}

// GetTree returns all categories as a forest (top-level categories with their
// nested children). The whole dataset is loaded in a single query and
// assembled in memory to avoid N+1 lookups.
func (m *Manager) GetTree() ([]*models.Category, error) {
	var cats []models.Category
	if err := m.q.GetAllCategories.Select(&cats); err != nil {
		m.lo.Error("error fetching knowledge categories", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	byID := make(map[int64]*models.Category, len(cats))
	for i := range cats {
		cats[i].Children = make([]*models.Category, 0)
		byID[cats[i].ID] = &cats[i]
	}
	roots := make([]*models.Category, 0)
	for i := range cats {
		c := &cats[i]
		if c.ParentID == nil {
			roots = append(roots, c)
			continue
		}
		if parent, ok := byID[*c.ParentID]; ok {
			parent.Children = append(parent.Children, c)
			continue
		}
		// Orphaned category (parent missing): surface it at the root so
		// the UI can still manage it.
		roots = append(roots, c)
	}
	return roots, nil
}

// CreateCategory creates a new category under the given parent (nil for a
// top-level "知识库" group).
func (m *Manager) CreateCategory(parentID *int64, code *string, name string) (models.Category, error) {
	var category models.Category
	var nextSort int
	if err := m.q.NextCategorySortOrder.Get(&nextSort, parentID); err != nil {
		m.lo.Error("error computing next knowledge category sort order", "error", err)
		return category, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := m.q.InsertCategory.Get(&category, parentID, code, name, nextSort); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return category, envelope.NewError(envelope.ConflictError, m.i18n.T("errors.alreadyExistsKnowledgeCategory"), nil)
		}
		m.lo.Error("error inserting knowledge category", "error", err)
		return category, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	category.Children = make([]*models.Category, 0)
	return category, nil
}

// UpdateCategory renames, re-parents, re-codes and/or reorders a category.
func (m *Manager) UpdateCategory(id int, parentID *int64, code *string, name string, sortOrder int) (models.Category, error) {
	var category models.Category

	current, err := m.getCategory(id)
	if err != nil {
		return category, err
	}
	if parentID == nil {
		parentID = current.ParentID
	}
	if parentID != nil {
		if *parentID == int64(id) {
			return category, envelope.NewError(envelope.InputError, m.i18n.T("errors.invalidKnowledgeCategoryMove"), nil)
		}
		cyclic, err := m.isDescendant(int64(id), *parentID)
		if err != nil {
			return category, err
		}
		if cyclic {
			return category, envelope.NewError(envelope.InputError, m.i18n.T("errors.invalidKnowledgeCategoryMove"), nil)
		}
	}

	if err := m.q.UpdateCategory.Get(&category, id, parentID, code, name, sortOrder); err != nil {
		if err == sql.ErrNoRows {
			return category, envelope.NewError(envelope.InputError, m.i18n.T("validation.notFoundKnowledgeCategory"), nil)
		}
		if dbutil.IsUniqueViolationError(err) {
			return category, envelope.NewError(envelope.ConflictError, m.i18n.T("errors.alreadyExistsKnowledgeCategory"), nil)
		}
		m.lo.Error("error updating knowledge category", "error", err)
		return category, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return category, nil
}

// DeleteCategory deletes a category (and, by cascade, its sub-categories
// and the articles attached to those sub-categories via FK).
func (m *Manager) DeleteCategory(id int) error {
	if _, err := m.q.DeleteCategory.Exec(id); err != nil {
		m.lo.Error("error deleting knowledge category", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

// CategoryHasArticles reports whether a category (or any of its sub-categories)
// still references knowledge articles. Callers can use this to refuse
// destructive UI actions.
func (m *Manager) CategoryHasArticles(id int) (bool, error) {
	var row struct {
		DirectCount int `db:"direct_count"`
	}
	if err := m.q.CountArticlesByCategory.Get(&row, id); err != nil {
		m.lo.Error("error counting knowledge articles by category", "error", err)
		return false, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return row.DirectCount > 0, nil
}

// GetArticle returns a single knowledge article with its category names and
// attachments.
func (m *Manager) GetArticle(id int) (models.Article, error) {
	var article models.Article
	if err := m.q.GetArticle.Get(&article, id); err != nil {
		if err == sql.ErrNoRows {
			return article, envelope.NewError(envelope.InputError, m.i18n.T("validation.notFoundKnowledgeArticle"), nil)
		}
		m.lo.Error("error fetching knowledge article", "error", err)
		return article, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	var attachments []models.Attachment
	if err := m.q.GetArticleAttachments.Select(&attachments, id); err != nil {
		m.lo.Error("error fetching knowledge article attachments", "error", err)
		return article, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if attachments == nil {
		attachments = []models.Attachment{}
	}
	article.Attachments = attachments
	return article, nil
}

// List returns a paginated slice of knowledge articles matching `f`.
func (m *Manager) List(f ListFilter) (models.ArticleListResult, error) {
	var result models.ArticleListResult

	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 {
		f.PerPage = 20
	}
	if f.PerPage > 200 {
		f.PerPage = 200
	}
	if f.SortOrder == "" {
		f.SortOrder = SortDesc
	}
	if f.KeywordScope == "" {
		f.KeywordScope = KeywordScopeAll
	}
	keywords := normalizeKeywords(f.Keywords)
	var keywordScope interface{}
	if len(keywords) == 0 {
		keywordScope = nil
	} else {
		keywordScope = f.KeywordScope
	}

	statusArg := interface{}(nil)
	if f.Status != nil {
		statusArg = string(*f.Status)
	}
	if f.DateField != "updated_at" {
		f.DateField = "created_at"
	}

	if err := m.q.CountArticles.Get(&result.Total,
		f.CategoryID, f.ParentCategoryID, statusArg, f.DateFrom, f.DateTo,
		keywordScope, f.KeywordScope, pq.Array(keywords),
		f.DateField,
	); err != nil {
		m.lo.Error("error counting knowledge articles", "error", err)
		return result, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	offset := (f.Page - 1) * f.PerPage
	if err := m.q.ListArticles.Select(&result.Articles,
		f.CategoryID, f.ParentCategoryID, statusArg, f.DateFrom, f.DateTo,
		keywordScope, f.KeywordScope, pq.Array(keywords),
		f.SortOrder, f.PerPage, offset,
		f.DateField,
	); err != nil {
		m.lo.Error("error listing knowledge articles", "error", err)
		return result, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	result.Page = f.Page
	result.PerPage = f.PerPage
	return result, nil
}

// CreateArticleInput is the payload used to create a new knowledge article.
type CreateArticleInput struct {
	CategoryID  *int64
	ExternalID  *string
	Code        *string
	Title       string
	Author      *string
	Content     string
	Status      models.ArticleStatus
	Attachments []models.AttachmentInput
}

// CreateArticle creates a new knowledge article along with its attachments.
func (m *Manager) CreateArticle(in CreateArticleInput) (models.Article, error) {
	var article models.Article
	if in.Status == "" {
		in.Status = models.ArticleStatusPublished
	}

	tx, err := m.db.Beginx()
	if err != nil {
		m.lo.Error("error starting knowledge article create transaction", "error", err)
		return article, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := tx.Stmtx(m.q.InsertArticle).Get(&article,
		in.CategoryID, in.ExternalID, in.Code, in.Title, in.Author, in.Content, string(in.Status), 0,
	); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return article, envelope.NewError(envelope.ConflictError, m.i18n.T("errors.alreadyExistsKnowledgeArticle"), nil)
		}
		if dbutil.IsForeignKeyError(err) {
			return article, envelope.NewError(envelope.InputError, m.i18n.T("validation.notFoundKnowledgeCategory"), nil)
		}
		m.lo.Error("error inserting knowledge article", "error", err)
		return article, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := insertAttachments(tx, m.q.InsertArticleAttachment, article.ID, in.Attachments); err != nil {
		return article, err
	}
	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing knowledge article create transaction", "error", err)
		return article, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return article, nil
}

// UpdateArticleInput is the payload used to update an existing knowledge article.
type UpdateArticleInput struct {
	CategoryID  *int64
	ExternalID  *string
	Code        *string
	Title       string
	Author      *string
	Content     string
	Status      models.ArticleStatus
	SortOrder   int
	Attachments []models.AttachmentInput
}

// UpdateArticle updates an existing knowledge article and replaces its
// attachment set.
func (m *Manager) UpdateArticle(id int, in UpdateArticleInput) (models.Article, error) {
	var article models.Article

	tx, err := m.db.Beginx()
	if err != nil {
		m.lo.Error("error starting knowledge article update transaction", "error", err)
		return article, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer tx.Rollback() //nolint:errcheck

	if err := tx.Stmtx(m.q.UpdateArticle).Get(&article,
		id, in.CategoryID, in.ExternalID, in.Code, in.Title, in.Author, in.Content, string(in.Status), in.SortOrder,
	); err != nil {
		if err == sql.ErrNoRows {
			return article, envelope.NewError(envelope.InputError, m.i18n.T("validation.notFoundKnowledgeArticle"), nil)
		}
		if dbutil.IsUniqueViolationError(err) {
			return article, envelope.NewError(envelope.ConflictError, m.i18n.T("errors.alreadyExistsKnowledgeArticle"), nil)
		}
		if dbutil.IsForeignKeyError(err) {
			return article, envelope.NewError(envelope.InputError, m.i18n.T("validation.notFoundKnowledgeCategory"), nil)
		}
		m.lo.Error("error updating knowledge article", "error", err)
		return article, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if _, err := tx.Stmtx(m.q.DeleteArticleAttachments).Exec(id); err != nil {
		m.lo.Error("error clearing knowledge article attachments", "error", err)
		return article, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := insertAttachments(tx, m.q.InsertArticleAttachment, int64(id), in.Attachments); err != nil {
		return article, err
	}
	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing knowledge article update transaction", "error", err)
		return article, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return article, nil
}

// DeleteArticle removes a knowledge article by id. Attachments are removed
// by FK cascade.
func (m *Manager) DeleteArticle(id int) error {
	if _, err := m.q.DeleteArticle.Exec(id); err != nil {
		m.lo.Error("error deleting knowledge article", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

// ListAllForExport returns every knowledge article flattened for the bulk
// exporter. The category columns are denormalised to ease the spreadsheet
// output.
func (m *Manager) ListAllForExport() ([]ExportRow, error) {
	type row struct {
		ID                 int64   `db:"id"`
		ExternalID         *string `db:"external_id"`
		Code               *string `db:"code"`
		Title              string  `db:"title"`
		Author             *string `db:"author"`
		Status             string  `db:"status"`
		CreatedAt          time.Time `db:"created_at"`
		UpdatedAt          time.Time `db:"updated_at"`
		ParentCategoryName string  `db:"parent_category_name"`
		CategoryName       string  `db:"category_name"`
		Content            string  `db:"content"`
	}
	var rows []row
	if err := m.q.ListAllArticles.Select(&rows); err != nil {
		m.lo.Error("error listing knowledge articles for export", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	out := make([]ExportRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, ExportRow{
			ID:                 r.ID,
			ExternalID:         r.ExternalID,
			Code:               r.Code,
			Title:              r.Title,
			Author:             r.Author,
			Status:             r.Status,
			CreatedAt:          r.CreatedAt,
			UpdatedAt:          r.UpdatedAt,
			ParentCategoryName: r.ParentCategoryName,
			CategoryName:       r.CategoryName,
			Content:            r.Content,
		})
	}
	return out, nil
}

// ExportRow is the denormalised article shape used by the bulk exporter.
// The JSON tags are snake_case so the client-side spreadsheet builder can
// read them by name.
type ExportRow struct {
	ID                 int64     `json:"id"`
	ExternalID         *string   `json:"external_id"`
	Code               *string   `json:"code"`
	Title              string    `json:"title"`
	Author             *string   `json:"author"`
	Status             string    `json:"status"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	ParentCategoryName string    `json:"parent_category_name"`
	CategoryName       string    `json:"category_name"`
	Content            string    `json:"content"`
}

// ImportItem represents a single row in the bulk-import spreadsheet.
// CategoryName may carry a combined "一级/二级" path (separated by "/" or
// "／"); in that case the first segment is treated as the level-1 category
// name and the rest as the level-2 category name. The same split applies
// to CategoryCode. Explicit Parent* fields (when provided by the client)
// take precedence.
type ImportItem struct {
	CategoryCode       string `json:"category_code"`
	CategoryName       string `json:"category_name"`
	ParentCategoryCode string `json:"parent_category_code,omitempty"`
	ParentCategoryName string `json:"parent_category_name,omitempty"`
	ExternalID         string `json:"external_id"`
	ArticleCode        string `json:"article_code"`
	Title              string `json:"title"`
	Author             string `json:"author"`
	Content            string `json:"content,omitempty"`
}

// splitCategoryPath splits a combined "一级/二级" category value on the
// first "/" (or fullwidth "／") separator. It returns the level-1 part and
// the level-2 part (the remainder joined back with "/"). A value without
// any separator yields only the first return value.
func splitCategoryPath(v string) (string, string) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", ""
	}
	normalized := strings.ReplaceAll(v, "／", "/")
	if idx := strings.Index(normalized, "/"); idx >= 0 {
		parent := strings.TrimSpace(normalized[:idx])
		child := strings.TrimSpace(normalized[idx+1:])
		return parent, child
	}
	return strings.TrimSpace(normalized), ""
}

// ImportError captures a per-row failure surfaced to the UI. Line numbers
// are 1-based and refer to the spreadsheet (header is line 1).
type ImportError struct {
	Line   int    `json:"line"`
	Reason string `json:"reason"`
}

// ImportStats summarises the outcome of a bulk import.
type ImportStats struct {
	CreatedCategories int           `json:"created_categories"`
	CreatedArticles   int           `json:"created_articles"`
	UpdatedArticles   int           `json:"updated_articles"`
	Errors            []ImportError `json:"errors"`
}

// Import upserts a batch of knowledge articles:
//   - Each row is matched by `external_id` (the spreadsheet "知识ID"). When a
//     row is matched, the existing article is updated; otherwise a new
//     article is created.
//   - Each row's category is matched by the (parent_code + name) pair. The
//     parent (level-1) is matched by code only, and the child (level-2) is
//     matched by parent + name. Missing categories are auto-created. A row
//     belonging to more than one category for the same `external_id` is
//     rejected.
//   - Title and author length are validated up front; rows that fail
//     validation are skipped and reported in `Errors`, the rest of the
//     batch still runs.
//
// The whole import runs inside a single database transaction so a failed
// import never leaves partial writes.
func (m *Manager) Import(items []ImportItem) (ImportStats, error) {
	var stats ImportStats

	tx, err := m.db.Beginx()
	if err != nil {
		m.lo.Error("error starting knowledge import transaction", "error", err)
		return stats, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer tx.Rollback() //nolint:errcheck

	// Cache categories for O(1) lookups: level-1 by name and by code,
	// level-2 by (parentID, name).
	parentByName := make(map[string]int64)
	parentByCode := make(map[string]int64)
	type childKey struct {
		ParentID int64
		Name     string
	}
	childByKey := make(map[childKey]int64)
	var allCats []models.Category
	if err := tx.Stmtx(m.q.GetAllCategories).Select(&allCats); err != nil {
		m.lo.Error("error fetching knowledge categories during import", "error", err)
		return stats, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	for _, c := range allCats {
		if c.ParentID == nil {
			parentByName[c.Name] = c.ID
			if c.Code != nil && *c.Code != "" {
				parentByCode[*c.Code] = c.ID
			}
			continue
		}
		childByKey[childKey{ParentID: *c.ParentID, Name: c.Name}] = c.ID
	}

	// Index existing articles by external_id for O(1) upserts.
	articleByExternal := make(map[string]int64)
	var existingExternal []models.Article
	q, args, err := sqlx.In(`select id, external_id, category_id from kb_articles where external_id is not null`)
	if err != nil {
		m.lo.Error("error building external-id lookup", "error", err)
		return stats, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	q = tx.Rebind(q)
	if err := tx.Select(&existingExternal, q, args...); err != nil {
		m.lo.Error("error fetching knowledge articles during import", "error", err)
		return stats, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	for _, a := range existingExternal {
		if a.ExternalID != nil {
			articleByExternal[*a.ExternalID] = a.ID
		}
	}

	for idx, it := range items {
		line := idx + 2 // +1 for zero-index, +1 for the header row
		title := strings.TrimSpace(it.Title)
		author := strings.TrimSpace(it.Author)
		externalID := strings.TrimSpace(it.ExternalID)
		articleCode := strings.TrimSpace(it.ArticleCode)
		content := strings.TrimSpace(it.Content)

		// Split the combined "一级/二级" category values. Explicit parent
		// fields from the client win. Otherwise the Category* fields are
		// treated as combined paths only when they actually contain a "/"
		// separator; without one the legacy template semantics apply
		// (分类Code = level-1 code, 分类名 = level-2 name).
		parentName, parentCode := strings.TrimSpace(it.ParentCategoryName), strings.TrimSpace(it.ParentCategoryCode)
		childName, childCode := strings.TrimSpace(it.CategoryName), strings.TrimSpace(it.CategoryCode)
		if parentName == "" && parentCode == "" {
			if strings.ContainsAny(it.CategoryName, "/／") || strings.ContainsAny(it.CategoryCode, "/／") {
				pName, cName := splitCategoryPath(it.CategoryName)
				pCode, cCode := splitCategoryPath(it.CategoryCode)
				parentName, childName = pName, cName
				parentCode, childCode = pCode, cCode
			} else {
				// Legacy layout: the code column identifies the level-1.
				parentCode = childCode
				childCode = ""
			}
		}
		// A code that is only a separator carries no information.
		if parentCode == "/" {
			parentCode = ""
		}
		if childCode == "/" {
			childCode = ""
		}

		if title == "" {
			stats.Errors = append(stats.Errors, ImportError{Line: line, Reason: "标题不能为空"})
			continue
		}
		if utf8.RuneCountInString(title) > MaxTitleLength {
			stats.Errors = append(stats.Errors, ImportError{Line: line, Reason: "标题超过最大长度"})
			continue
		}
		if parentName == "" && childName == "" && parentCode == "" {
			stats.Errors = append(stats.Errors, ImportError{Line: line, Reason: "分类名不能为空"})
			continue
		}
		nameTooLong := false
		for _, n := range []string{parentName, childName} {
			if utf8.RuneCountInString(n) > MaxCategoryNameLength {
				nameTooLong = true
				break
			}
		}
		if nameTooLong {
			stats.Errors = append(stats.Errors, ImportError{Line: line, Reason: "分类名超过最大长度"})
			continue
		}

		// Resolve / create the level-1 (parent) category: by code first,
		// then by name; created with the name (falling back to the code).
		var parentID *int64
		if parentName != "" || parentCode != "" {
			var id int64
			var ok bool
			if parentCode != "" {
				id, ok = parentByCode[parentCode]
			}
			if !ok && parentName != "" {
				id, ok = parentByName[parentName]
			}
			if !ok {
				createName := parentName
				if createName == "" {
					createName = parentCode
				}
				var codePtr *string
				if parentCode != "" {
					codePtr = &parentCode
				}
				var newID int64
				if err := tx.Stmtx(m.q.InsertCategoryOnConflict).QueryRowx(nil, codePtr, createName, 0).Scan(&newID); err != nil {
					if errors.Is(err, sql.ErrNoRows) {
						// Conflict on (parent_id, name); load the existing row.
						var existing models.Category
						if err2 := tx.Stmtx(m.q.GetCategoryByParentAndName).Get(&existing, nil, createName); err2 == nil {
							newID = existing.ID
						} else {
							m.lo.Error("error recovering parent category after conflict", "error", err2)
							stats.Errors = append(stats.Errors, ImportError{Line: line, Reason: "一级分类创建失败"})
							continue
						}
					} else {
						m.lo.Error("error inserting knowledge parent category during import", "error", err)
						stats.Errors = append(stats.Errors, ImportError{Line: line, Reason: "一级分类创建失败"})
						continue
					}
				}
				stats.CreatedCategories++
				id = newID
				parentByName[createName] = newID
				if parentCode != "" {
					parentByCode[parentCode] = newID
				}
			}
			pid := id
			parentID = &pid
		}

		// Resolve / create the level-2 (child) category by (parentID, name).
		// When no child name is given the article attaches directly to the
		// level-1 category.
		var catID int64
		if childName != "" {
			var ok bool
			if parentID != nil {
				catID, ok = childByKey[childKey{ParentID: *parentID, Name: childName}]
			}
			if !ok {
				var codePtr *string
				if childCode != "" {
					codePtr = &childCode
				}
				var newID int64
				if err := tx.Stmtx(m.q.InsertCategoryOnConflict).QueryRowx(parentID, codePtr, childName, 0).Scan(&newID); err != nil {
					if errors.Is(err, sql.ErrNoRows) {
						var existing models.Category
						if err2 := tx.Stmtx(m.q.GetCategoryByParentAndName).Get(&existing, parentID, childName); err2 == nil {
							newID = existing.ID
						} else {
							m.lo.Error("error recovering child category after conflict", "error", err2)
							stats.Errors = append(stats.Errors, ImportError{Line: line, Reason: "二级分类创建失败"})
							continue
						}
					} else {
						m.lo.Error("error inserting knowledge child category during import", "error", err)
						stats.Errors = append(stats.Errors, ImportError{Line: line, Reason: "二级分类创建失败"})
						continue
					}
				}
				stats.CreatedCategories++
				catID = newID
				if parentID != nil {
					childByKey[childKey{ParentID: *parentID, Name: childName}] = newID
				}
			}
		} else if parentID != nil {
			catID = *parentID
		}

		// Refuse multi-category imports for the same external_id.
		if existingID, exists := articleByExternal[externalID]; exists && externalID != "" {
			var row struct {
				Conflict bool `db:"conflict"`
			}
			cid := catID
			if err := tx.Stmtx(m.q.ArticleExistsInOtherCategories).Get(&row, externalID, existingID, &cid); err == nil && row.Conflict {
				stats.Errors = append(stats.Errors, ImportError{Line: line, Reason: "知识ID已存在且属于其他分类"})
				continue
			}
		}

		var extIDPtr, codePtr, authorPtr *string
		if externalID != "" {
			extIDPtr = &externalID
		}
		if articleCode != "" {
			codePtr = &articleCode
		}
		if author != "" {
			authorPtr = &author
		}
		cid := catID
		if existingID, exists := articleByExternal[externalID]; exists && externalID != "" {
			if _, err := tx.Stmtx(m.q.UpdateArticle).Exec(existingID, &cid, extIDPtr, codePtr, title, authorPtr, content, string(models.ArticleStatusPublished), 0); err != nil {
				if dbutil.IsUniqueViolationError(err) {
					stats.Errors = append(stats.Errors, ImportError{Line: line, Reason: "知识Code 已存在"})
					continue
				}
				m.lo.Error("error updating knowledge article during import", "error", err)
				stats.Errors = append(stats.Errors, ImportError{Line: line, Reason: "更新知识失败"})
				continue
			}
			stats.UpdatedArticles++
		} else {
			var newID int64
			if err := tx.Stmtx(m.q.InsertArticleOnConflict).QueryRowx(&cid, extIDPtr, codePtr, title, authorPtr, content, string(models.ArticleStatusPublished), 0).Scan(&newID); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					// A different article already holds this external_id; load it
					// and update in place so the importer remains idempotent.
					var existing models.Article
					if err2 := tx.Stmtx(m.q.GetArticleByExternalID).Get(&existing, externalID); err2 == nil {
						if _, err3 := tx.Stmtx(m.q.UpdateArticle).Exec(existing.ID, &cid, extIDPtr, codePtr, title, authorPtr, content, string(models.ArticleStatusPublished), 0); err3 != nil {
							if dbutil.IsUniqueViolationError(err3) {
								stats.Errors = append(stats.Errors, ImportError{Line: line, Reason: "知识Code 已存在"})
								continue
							}
							m.lo.Error("error updating existing knowledge article during import", "error", err3)
							stats.Errors = append(stats.Errors, ImportError{Line: line, Reason: "更新知识失败"})
							continue
						}
						stats.UpdatedArticles++
						articleByExternal[externalID] = existing.ID
						continue
					}
					stats.Errors = append(stats.Errors, ImportError{Line: line, Reason: "知识Code 已存在"})
					continue
				}
				if dbutil.IsUniqueViolationError(err) {
					stats.Errors = append(stats.Errors, ImportError{Line: line, Reason: "知识Code 已存在"})
					continue
				}
				m.lo.Error("error inserting knowledge article during import", "error", err)
				stats.Errors = append(stats.Errors, ImportError{Line: line, Reason: "新增知识失败"})
				continue
			}
			stats.CreatedArticles++
			if externalID != "" {
				articleByExternal[externalID] = newID
			}
		}
	}

	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing knowledge import transaction", "error", err)
		return stats, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return stats, nil
}

// getCategory fetches a single category by ID.
func (m *Manager) getCategory(id int) (models.Category, error) {
	var category models.Category
	if err := m.q.GetCategory.Get(&category, id); err != nil {
		if err == sql.ErrNoRows {
			return category, envelope.NewError(envelope.InputError, m.i18n.T("validation.notFoundKnowledgeCategory"), nil)
		}
		m.lo.Error("error fetching knowledge category", "error", err)
		return category, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return category, nil
}

// isDescendant reports whether `candidate` is a (transitive) child of
// `ancestor`. Used to prevent moving a category under one of its own
// descendants.
func (m *Manager) isDescendant(ancestor, candidate int64) (bool, error) {
	var cats []models.Category
	if err := m.q.GetAllCategories.Select(&cats); err != nil {
		m.lo.Error("error fetching knowledge categories", "error", err)
		return false, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	parentOf := make(map[int64]*int64, len(cats))
	for i := range cats {
		parentOf[cats[i].ID] = cats[i].ParentID
	}
	seen := make(map[int64]bool)
	cur := candidate
	for {
		if cur == ancestor {
			return true, nil
		}
		if seen[cur] {
			return false, nil
		}
		seen[cur] = true
		parent, ok := parentOf[cur]
		if !ok || parent == nil {
			return false, nil
		}
		cur = *parent
	}
}

// insertAttachments is a small helper that writes the supplied attachments
// for a given article. It is shared between Create and Update.
func insertAttachments(tx *sqlx.Tx, stmt *sqlx.Stmt, articleID int64, in []models.AttachmentInput) error {
	for _, a := range in {
		uuid := strings.TrimSpace(a.MediaUUID)
		if uuid == "" {
			continue
		}
		if _, err := tx.Stmtx(stmt).Exec(articleID, uuid, a.FileName, a.ContentType, a.Size); err != nil {
			return envelope.NewError(envelope.GeneralError, "globals.messages.somethingWentWrong", nil)
		}
	}
	return nil
}

// normalizeKeywords trims, lower-cases and de-duplicates the user-supplied
// keywords. Empty strings are dropped. Returns nil if no usable keywords
// remain so callers can skip the keyword filter entirely.
func normalizeKeywords(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, k := range in {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
