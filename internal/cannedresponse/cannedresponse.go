// Package cannedresponse handles the management of globally shared canned
// responses ("公共常用语"). Unlike the per-inbox quick replies, canned responses
// are global and are grouped under arbitrarily nested categories.
package cannedresponse

import (
	"database/sql"
	"embed"
	"strconv"
	"strings"

	"github.com/abhinavxd/libredesk/internal/cannedresponse/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/zerodha/logf"
)

// Length limits mirror the constraints enforced by the download template and
// the database CHECK constraints.
const (
	MaxCategoryNameLength = 20
	MaxResponseLength     = 1000
)

var (
	//go:embed queries.sql
	efs embed.FS
)

// Manager manages canned response categories and entries.
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

// queries contains prepared SQL queries.
type queries struct {
	GetAllCategories      *sqlx.Stmt `query:"get-all-categories"`
	GetCategory           *sqlx.Stmt `query:"get-category"`
	InsertCategory        *sqlx.Stmt `query:"insert-category"`
	UpdateCategory        *sqlx.Stmt `query:"update-category"`
	DeleteCategory        *sqlx.Stmt `query:"delete-category"`
	NextCategorySortOrder *sqlx.Stmt `query:"next-category-sort-order"`
	GetAllResponses       *sqlx.Stmt `query:"get-all-responses"`
	GetResponse           *sqlx.Stmt `query:"get-response"`
	InsertResponse        *sqlx.Stmt `query:"insert-response"`
	UpdateResponse        *sqlx.Stmt `query:"update-response"`
	DeleteResponse        *sqlx.Stmt `query:"delete-response"`
	NextResponseSortOrder *sqlx.Stmt `query:"next-response-sort-order"`
	DeleteAllResponses    *sqlx.Stmt `query:"delete-all-responses"`
	DeleteAllCategories   *sqlx.Stmt `query:"delete-all-categories"`
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

// GetTree returns all categories as a forest (top-level categories with their
// nested children and responses attached). The whole dataset is loaded in two
// queries and assembled in memory to avoid N+1 lookups.
func (m *Manager) GetTree() ([]*models.Category, error) {
	var cats []models.Category
	if err := m.q.GetAllCategories.Select(&cats); err != nil {
		m.lo.Error("error fetching canned categories", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	var responses []models.Response
	if err := m.q.GetAllResponses.Select(&responses); err != nil {
		m.lo.Error("error fetching canned responses", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	byID := make(map[int64]*models.Category, len(cats))
	for i := range cats {
		cats[i].Children = make([]*models.Category, 0)
		cats[i].Responses = make([]models.Response, 0)
		byID[cats[i].ID] = &cats[i]
	}
	for _, r := range responses {
		if c, ok := byID[r.CategoryID]; ok {
			c.Responses = append(c.Responses, r)
		}
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
		// Orphaned category (parent missing): surface it at the root.
		roots = append(roots, c)
	}
	return roots, nil
}

// CreateCategory creates a new category under the given parent (nil for a
// top-level category).
func (m *Manager) CreateCategory(parentID *int64, name string) (models.Category, error) {
	var category models.Category
	var nextSort int
	if err := m.q.NextCategorySortOrder.Get(&nextSort, parentID); err != nil {
		m.lo.Error("error computing next canned category sort order", "error", err)
		return category, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := m.q.InsertCategory.Get(&category, parentID, name, nextSort); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return category, envelope.NewError(envelope.ConflictError, m.i18n.T("errors.alreadyExistsCannedCategory"), nil)
		}
		if dbutil.IsForeignKeyError(err) {
			return category, envelope.NewError(envelope.InputError, m.i18n.T("validation.notFoundCannedCategory"), nil)
		}
		m.lo.Error("error inserting canned category", "error", err)
		return category, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	category.Children = make([]*models.Category, 0)
	category.Responses = make([]models.Response, 0)
	return category, nil
}

// UpdateCategory renames, re-parents and/or reorders a category.
func (m *Manager) UpdateCategory(id int, parentID *int64, name string, sortOrder int) (models.Category, error) {
	var category models.Category

	current, err := m.getCategory(id)
	if err != nil {
		return category, err
	}

	// Keep the existing parent when the caller does not specify one.
	if parentID == nil {
		parentID = current.ParentID
	}
	if parentID != nil {
		if *parentID == int64(id) {
			return category, envelope.NewError(envelope.InputError, m.i18n.T("errors.invalidCannedCategoryMove"), nil)
		}
		cyclic, err := m.isDescendant(int64(id), *parentID)
		if err != nil {
			return category, err
		}
		if cyclic {
			return category, envelope.NewError(envelope.InputError, m.i18n.T("errors.invalidCannedCategoryMove"), nil)
		}
	}

	if err := m.q.UpdateCategory.Get(&category, id, parentID, name, sortOrder); err != nil {
		if err == sql.ErrNoRows {
			return category, envelope.NewError(envelope.InputError, m.i18n.T("validation.notFoundCannedCategory"), nil)
		}
		if dbutil.IsUniqueViolationError(err) {
			return category, envelope.NewError(envelope.ConflictError, m.i18n.T("errors.alreadyExistsCannedCategory"), nil)
		}
		if dbutil.IsForeignKeyError(err) {
			return category, envelope.NewError(envelope.InputError, m.i18n.T("validation.notFoundCannedCategory"), nil)
		}
		m.lo.Error("error updating canned category", "error", err)
		return category, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return category, nil
}

// DeleteCategory deletes a category. Nested sub-categories and responses are
// removed by the database via ON DELETE CASCADE.
func (m *Manager) DeleteCategory(id int) error {
	if _, err := m.q.DeleteCategory.Exec(id); err != nil {
		m.lo.Error("error deleting canned category", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

// CreateResponse creates a new canned response under the given category.
func (m *Manager) CreateResponse(categoryID int, content string) (models.Response, error) {
	var response models.Response
	var nextSort int
	if err := m.q.NextResponseSortOrder.Get(&nextSort, categoryID); err != nil {
		m.lo.Error("error computing next canned response sort order", "error", err)
		return response, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := m.q.InsertResponse.Get(&response, categoryID, content, nextSort); err != nil {
		if dbutil.IsForeignKeyError(err) {
			return response, envelope.NewError(envelope.InputError, m.i18n.T("validation.notFoundCannedCategory"), nil)
		}
		m.lo.Error("error inserting canned response", "error", err)
		return response, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return response, nil
}

// UpdateResponse updates the content and sort order of a canned response.
func (m *Manager) UpdateResponse(id int, content string, sortOrder int) (models.Response, error) {
	var response models.Response
	if err := m.q.UpdateResponse.Get(&response, id, content, sortOrder); err != nil {
		if err == sql.ErrNoRows {
			return response, envelope.NewError(envelope.InputError, m.i18n.T("validation.notFoundCannedResponse"), nil)
		}
		m.lo.Error("error updating canned response", "error", err)
		return response, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return response, nil
}

// DeleteResponse deletes a single canned response by ID.
func (m *Manager) DeleteResponse(id int) error {
	if _, err := m.q.DeleteResponse.Exec(id); err != nil {
		m.lo.Error("error deleting canned response", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

// DeleteAll removes every category and canned response.
func (m *Manager) DeleteAll() error {
	tx, err := m.db.Beginx()
	if err != nil {
		m.lo.Error("error starting canned response delete-all transaction", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.Stmtx(m.q.DeleteAllResponses).Exec(); err != nil {
		m.lo.Error("error deleting all canned responses", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if _, err := tx.Stmtx(m.q.DeleteAllCategories).Exec(); err != nil {
		m.lo.Error("error deleting all canned categories", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing canned response delete-all transaction", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

// ImportItem represents a single category (with its responses and nested
// sub-categories) in a bulk import payload.
type ImportItem struct {
	Name      string       `json:"name"`
	Responses []string     `json:"responses"`
	Children  []ImportItem `json:"children"`
}

// ImportStats reports how many categories and responses were created or
// skipped during a bulk import.
type ImportStats struct {
	CreatedCategories int `json:"created_categories"`
	CreatedResponses  int `json:"created_responses"`
	SkippedResponses  int `json:"skipped_responses"`
}

// Import bulk-imports a category tree using an append-merge strategy:
//   - A category is matched by (parent, name). When it matches an existing
//     category, that category is reused and the import recurses into it.
//   - A category that matches nothing is created.
//   - Responses are appended to their category, skipping ones whose content
//     already exists in that category. No existing data is ever deleted.
//
// The whole import runs inside a single database transaction so a failed
// import never leaves partial writes.
func (m *Manager) Import(items []ImportItem) (ImportStats, error) {
	var stats ImportStats
	tx, err := m.db.Beginx()
	if err != nil {
		m.lo.Error("error starting canned response import transaction", "error", err)
		return stats, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer tx.Rollback() //nolint:errcheck

	var cats []models.Category
	if err := tx.Stmtx(m.q.GetAllCategories).Select(&cats); err != nil {
		m.lo.Error("error fetching canned categories during import", "error", err)
		return stats, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	var responses []models.Response
	if err := tx.Stmtx(m.q.GetAllResponses).Select(&responses); err != nil {
		m.lo.Error("error fetching canned responses during import", "error", err)
		return stats, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	// Index existing data for O(1) lookups.
	catByKey := make(map[string]models.Category, len(cats))
	nextCatSort := make(map[int64]int)
	for _, c := range cats {
		catByKey[categoryKey(c.ParentID, c.Name)] = c
		p := int64(0)
		if c.ParentID != nil {
			p = *c.ParentID
		}
		if c.SortOrder+1 > nextCatSort[p] {
			nextCatSort[p] = c.SortOrder + 1
		}
	}
	responseSeen := make(map[int64]map[string]bool)
	nextResponseSort := make(map[int64]int)
	for _, r := range responses {
		if responseSeen[r.CategoryID] == nil {
			responseSeen[r.CategoryID] = make(map[string]bool)
		}
		responseSeen[r.CategoryID][r.Content] = true
		if r.SortOrder+1 > nextResponseSort[r.CategoryID] {
			nextResponseSort[r.CategoryID] = r.SortOrder + 1
		}
	}

	var process func(parentID *int64, its []ImportItem) error
	process = func(parentID *int64, its []ImportItem) error {
		for _, it := range its {
			name := strings.TrimSpace(it.Name)
			if name == "" {
				continue
			}
			key := categoryKey(parentID, name)
			cat, ok := catByKey[key]
			if !ok {
				p := int64(0)
				if parentID != nil {
					p = *parentID
				}
				sort := nextCatSort[p]
				var created models.Category
				if err := tx.Stmtx(m.q.InsertCategory).Get(&created, parentID, name, sort); err != nil {
					if dbutil.IsUniqueViolationError(err) {
						return envelope.NewError(envelope.ConflictError, m.i18n.T("errors.alreadyExistsCannedCategory"), nil)
					}
					m.lo.Error("error inserting canned category during import", "error", err)
					return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
				}
				cat = created
				catByKey[key] = created
				nextCatSort[p] = sort + 1
				stats.CreatedCategories++
			}

			if responseSeen[cat.ID] == nil {
				responseSeen[cat.ID] = make(map[string]bool)
			}
			for _, raw := range it.Responses {
				content := strings.TrimSpace(raw)
				if content == "" {
					continue
				}
				if responseSeen[cat.ID][content] {
					stats.SkippedResponses++
					continue
				}
				sort := nextResponseSort[cat.ID]
				if _, err := tx.Stmtx(m.q.InsertResponse).Exec(cat.ID, content, sort); err != nil {
					m.lo.Error("error inserting canned response during import", "error", err)
					return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
				}
				nextResponseSort[cat.ID] = sort + 1
				responseSeen[cat.ID][content] = true
				stats.CreatedResponses++
			}

			if len(it.Children) > 0 {
				childParent := cat.ID
				if err := process(&childParent, it.Children); err != nil {
					return err
				}
			}
		}
		return nil
	}

	if err := process(nil, items); err != nil {
		return stats, err
	}
	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing canned response import transaction", "error", err)
		return stats, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return stats, nil
}

// getCategory fetches a single category by ID.
func (m *Manager) getCategory(id int) (models.Category, error) {
	var category models.Category
	if err := m.q.GetCategory.Get(&category, id); err != nil {
		if err == sql.ErrNoRows {
			return category, envelope.NewError(envelope.InputError, m.i18n.T("validation.notFoundCannedCategory"), nil)
		}
		m.lo.Error("error fetching canned category", "error", err)
		return category, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return category, nil
}

// isDescendant reports whether `descendant` is a (transitive) child of
// `ancestor`. It is used to prevent moving a category under one of its own
// descendants.
func (m *Manager) isDescendant(ancestor, candidate int64) (bool, error) {
	var cats []models.Category
	if err := m.q.GetAllCategories.Select(&cats); err != nil {
		m.lo.Error("error fetching canned categories", "error", err)
		return false, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	parentOf := make(map[int64]*int64, len(cats))
	for i := range cats {
		parentOf[cats[i].ID] = cats[i].ParentID
	}
	// Walk up from the candidate towards the root looking for the ancestor.
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

// categoryKey builds a lookup key for a category identified by its parent and
// name. A nil parent (top-level) is encoded distinctly.
func categoryKey(parentID *int64, name string) string {
	if parentID == nil {
		return "\x00" + name
	}
	return strconv.FormatInt(*parentID, 10) + "\x00" + name
}
