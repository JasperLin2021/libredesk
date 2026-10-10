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

	// Per-agent canned responses ("个人常用语") and pins ("置顶常用语").
	GetUserPersonalResponses  *sqlx.Stmt `query:"get-user-personal-responses"`
	GetUserPersonalResponse   *sqlx.Stmt `query:"get-user-personal-response"`
	NextUserPersonalSortOrder *sqlx.Stmt `query:"next-user-personal-sort-order"`
	InsertUserPersonalResponse *sqlx.Stmt `query:"insert-user-personal-response"`
	UpdateUserPersonalResponse *sqlx.Stmt `query:"update-user-personal-response"`
	DeleteUserPersonalResponse *sqlx.Stmt `query:"delete-user-personal-response"`
	GetUserCannedPins         *sqlx.Stmt `query:"get-user-canned-pins"`
	InsertUserCannedPin       *sqlx.Stmt `query:"insert-user-canned-pin"`
	UpdateUserCannedPinSort   *sqlx.Stmt `query:"update-user-canned-pin-sort"`
	DeleteUserCannedPin       *sqlx.Stmt `query:"delete-user-canned-pin"`
	NextUserCannedPinSort     *sqlx.Stmt `query:"next-user-canned-pin-sort-order"`
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

// GetAgentView returns the aggregate view of canned responses shown in the
// agent's "常用语" sidebar tab. It includes:
//
//   - `Global`: the full global category tree (the same data the admin
//     "公共常用语" page edits). It is loaded in a single call to keep the
//     payload small and to avoid N+1 queries.
//   - `Personal`: the agent's own personal canned responses, scoped to
//     `userID` so they never see another agent's entries.
//   - `Pinned`: the agent's pin list, materialized with the resolved
//     content of each pinned response. Pins whose underlying response has
//     since been deleted (admin removed a global response, or the agent
//     removed a personal one) are silently dropped — a stale pin must
//     never leak into the UI.
//
// Two SQL round-trips load the global tree and the personal/pin data;
// everything else is assembled in memory.
func (m *Manager) GetAgentView(userID int) (models.AgentView, error) {
	var view models.AgentView

	// Reuse the existing admin tree builder. The whole tree is needed
	// anyway because the sidebar nests the same global categories under
	// the personal/pinned groups.
	roots, err := m.GetTree()
	if err != nil {
		return view, err
	}
	view.Global = roots

	personal, err := m.listPersonal(userID)
	if err != nil {
		return view, err
	}
	view.Personal = personal

	pins, err := m.listPins(userID)
	if err != nil {
		return view, err
	}
	view.Pinned = pins
	return view, nil
}

// CreatePersonal appends a new personal canned response owned by `userID`.
// The caller is expected to have already validated/truncated `content`.
func (m *Manager) CreatePersonal(userID int, content string) (models.PersonalResponse, error) {
	var resp models.PersonalResponse
	var nextSort int
	if err := m.q.NextUserPersonalSortOrder.Get(&nextSort, userID); err != nil {
		m.lo.Error("error computing next personal canned sort order", "error", err)
		return resp, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := m.q.InsertUserPersonalResponse.Get(&resp, userID, content, nextSort); err != nil {
		m.lo.Error("error inserting personal canned response", "error", err)
		return resp, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return resp, nil
}

// UpdatePersonal updates a personal canned response owned by `userID`. A
// mismatched owner id returns InputError so the handler can surface a 404
// instead of leaking the existence of someone else's entry.
func (m *Manager) UpdatePersonal(userID, id int, content string, sortOrder int) (models.PersonalResponse, error) {
	var resp models.PersonalResponse
	if err := m.q.UpdateUserPersonalResponse.Get(&resp, id, content, sortOrder, userID); err != nil {
		if err == sql.ErrNoRows {
			return resp, envelope.NewError(envelope.InputError, m.i18n.T("validation.notFoundCannedResponse"), nil)
		}
		m.lo.Error("error updating personal canned response", "error", err)
		return resp, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return resp, nil
}

// DeletePersonal removes a personal canned response owned by `userID`. The
// matching pin rows for the same personal response are cleaned up by a
// cascading FK on `user_canned_pins` (`user_id` is the FK), so no extra
// work is needed here.
func (m *Manager) DeletePersonal(userID, id int) error {
	if _, err := m.q.DeleteUserPersonalResponse.Exec(id, userID); err != nil {
		m.lo.Error("error deleting personal canned response", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

// Pin pins a canned response to the top of `userID`'s sidebar. The pin is
// idempotent: re-pinning the same response is a no-op rather than an error.
func (m *Manager) Pin(userID int, source models.PinSource, responseID int) error {
	if source != models.PinSourceGlobal && source != models.PinSourcePersonal {
		return envelope.NewError(envelope.InputError, m.i18n.T("validation.invalidValue"), nil)
	}
	var nextSort int
	if err := m.q.NextUserCannedPinSort.Get(&nextSort, userID); err != nil {
		m.lo.Error("error computing next pin sort order", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if _, err := m.q.InsertUserCannedPin.Exec(userID, string(source), responseID, nextSort); err != nil {
		m.lo.Error("error inserting canned pin", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

// Unpin removes a canned response from `userID`'s pin list. Missing pins
// are silently ignored to keep the operation idempotent for the UI.
func (m *Manager) Unpin(userID int, source models.PinSource, responseID int) error {
	if source != models.PinSourceGlobal && source != models.PinSourcePersonal {
		return envelope.NewError(envelope.InputError, m.i18n.T("validation.invalidValue"), nil)
	}
	if _, err := m.q.DeleteUserCannedPin.Exec(userID, string(source), responseID); err != nil {
		m.lo.Error("error deleting canned pin", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

// listPersonal returns the per-user personal canned responses. Centralized
// here so it can be reused by the manager without leaking the prepared stmt
// to callers.
func (m *Manager) listPersonal(userID int) ([]models.PersonalResponse, error) {
	var rows []models.PersonalResponse
	if err := m.q.GetUserPersonalResponses.Select(&rows, userID); err != nil {
		m.lo.Error("error fetching personal canned responses", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return rows, nil
}

// listPins returns the user's pin list, joined with the underlying canned
// response content. The join drops any pin whose target has been deleted
// (admin removed a global response, or the agent removed a personal one)
// rather than surfacing an empty row to the UI.
func (m *Manager) listPins(userID int) ([]models.PinnedResponse, error) {
	type pinRow struct {
		Source     string `db:"source"`
		ResponseID int64  `db:"response_id"`
		SortOrder  int    `db:"sort_order"`
	}
	var pins []pinRow
	if err := m.q.GetUserCannedPins.Select(&pins, userID); err != nil {
		m.lo.Error("error fetching canned pins", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if len(pins) == 0 {
		return []models.PinnedResponse{}, nil
	}

	// Bucket ids by source for a single SELECT per source.
	var globalIDs, personalIDs []int64
	for _, p := range pins {
		switch models.PinSource(p.Source) {
		case models.PinSourceGlobal:
			globalIDs = append(globalIDs, p.ResponseID)
		case models.PinSourcePersonal:
			personalIDs = append(personalIDs, p.ResponseID)
		}
	}

	globalContent := make(map[int64]string)
	if len(globalIDs) > 0 {
		q, args, err := sqlx.In(`select id, content from canned_responses where id in (?)`, globalIDs)
		if err != nil {
			m.lo.Error("error building canned pin join query", "error", err)
			return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
		q = m.db.Rebind(q)
		type row struct {
			ID      int64  `db:"id"`
			Content string `db:"content"`
		}
		var rows []row
		if err := m.db.Select(&rows, q, args...); err != nil {
			m.lo.Error("error joining canned pins to global responses", "error", err)
			return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
		for _, r := range rows {
			globalContent[r.ID] = r.Content
		}
	}

	personalContent := make(map[int64]string)
	if len(personalIDs) > 0 {
		q, args, err := sqlx.In(`select id, content from user_canned_responses where user_id = ? and id in (?)`, userID, personalIDs)
		if err != nil {
			m.lo.Error("error building personal canned pin join query", "error", err)
			return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
		q = m.db.Rebind(q)
		type row struct {
			ID      int64  `db:"id"`
			Content string `db:"content"`
		}
		var rows []row
		if err := m.db.Select(&rows, q, args...); err != nil {
			m.lo.Error("error joining canned pins to personal responses", "error", err)
			return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
		for _, r := range rows {
			personalContent[r.ID] = r.Content
		}
	}

	out := make([]models.PinnedResponse, 0, len(pins))
	for _, p := range pins {
		var content string
		switch models.PinSource(p.Source) {
		case models.PinSourceGlobal:
			content = globalContent[p.ResponseID]
		case models.PinSourcePersonal:
			content = personalContent[p.ResponseID]
		}
		if content == "" {
			// Stale pin: the underlying response was removed. Skip silently
			// so the UI never renders a blank row.
			continue
		}
		out = append(out, models.PinnedResponse{
			Source:     models.PinSource(p.Source),
			ResponseID: p.ResponseID,
			Content:    content,
			SortOrder:  p.SortOrder,
		})
	}
	return out, nil
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
