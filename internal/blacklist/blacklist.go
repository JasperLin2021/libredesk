// Package blacklist manages the "在线黑名单" (online blacklist) feature:
// contacts (and visitors) that an agent has blocked from the live chat widget.
// A blocked contact's new conversations are never persisted — the widget only
// ever receives the configured welcome message.
package blacklist

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/blacklist/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/zerodha/logf"
)

const (
	defaultPageSize = 30
	maxPageSize     = 100

	// Tabs of the customer list.
	FilterAll     = "all"
	FilterBlocked = "blocked"
)

var (
	//go:embed queries.sql
	efs embed.FS

	// Whitelisted sort columns so user input is never interpolated raw.
	customerSortColumns = map[string]string{
		"blacklisted_at": "b.created_at",
		"contact_name":   "contact_name",
		"actor_name":     "actor_name",
		"created_at":     "u.created_at",
	}
	logSortColumns = map[string]string{
		"created_at":   "l.created_at",
		"contact_name": "contact_name",
		"actor_name":   "actor_name",
	}
)

// Opts contains options for initializing the Manager.
type Opts struct {
	DB   *sqlx.DB
	Lo   *logf.Logger
	I18n *i18n.I18n
}

// queries contains prepared SQL queries.
type queries struct {
	GetBlacklist          *sqlx.Stmt `query:"get-blacklist"`
	UpsertBlacklist       *sqlx.Stmt `query:"upsert-blacklist"`
	DeleteBlacklist       *sqlx.Stmt `query:"delete-blacklist"`
	InsertBlacklistLog    *sqlx.Stmt `query:"insert-blacklist-log"`
	CountBlacklisted      *sqlx.Stmt `query:"count-blacklisted"`
	GetBlacklistCustomers string     `query:"get-blacklist-customers"`
	GetBlacklistLogs      string     `query:"get-blacklist-logs"`
}

// Manager exposes the blacklist operations.
type Manager struct {
	q    queries
	db   *sqlx.DB
	lo   *logf.Logger
	i18n *i18n.I18n
}

// CustomerFilters holds the list options of the customer ("客户信息") page.
type CustomerFilters struct {
	Filter   string
	Search   string
	Action   string
	From     *time.Time
	To       *time.Time
	Order    string
	OrderBy  string
	Page     int
	PageSize int
}

// LogFilters holds the list options of the operation log ("操作日志") page.
type LogFilters struct {
	Action   string
	Search   string
	From     *time.Time
	To       *time.Time
	Order    string
	OrderBy  string
	Page     int
	PageSize int
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

// Status returns the current blacklist state of a contact.
func (m *Manager) Status(contactID int) (models.Status, error) {
	if contactID <= 0 {
		return models.Status{}, nil
	}
	var b models.Blacklist
	if err := m.q.GetBlacklist.Get(&b, contactID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Status{}, nil
		}
		m.lo.Error("error fetching contact blacklist status", "contact_id", contactID, "error", err)
		return models.Status{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return models.Status{
		Blocked:   true,
		Reason:    b.Reason,
		CreatedBy: b.CreatedBy,
		CreatedAt: &b.CreatedAt,
	}, nil
}

// IsBlocked reports whether the contact is currently blacklisted.
func (m *Manager) IsBlocked(contactID int) (bool, error) {
	if contactID <= 0 {
		return false, nil
	}
	var b models.Blacklist
	if err := m.q.GetBlacklist.Get(&b, contactID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		m.lo.Error("error checking contact blacklist", "contact_id", contactID, "error", err)
		return false, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return true, nil
}

// Add blacklists a contact and appends an operation log entry. Re-adding a
// contact that is already blacklisted refreshes the reason and the actor.
func (m *Manager) Add(contactID, actorID int, reason string) error {
	if contactID <= 0 {
		return envelope.NewError(envelope.InputError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	tx, err := m.db.BeginTxx(context.Background(), &sql.TxOptions{})
	if err != nil {
		m.lo.Error("error starting transaction for blacklist add", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer tx.Rollback()

	if _, err := tx.Stmtx(m.q.UpsertBlacklist).Exec(contactID, reason, nullInt(actorID)); err != nil {
		m.lo.Error("error upserting contact blacklist", "contact_id", contactID, "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if _, err := tx.Stmtx(m.q.InsertBlacklistLog).Exec(contactID, models.ActionAdd, reason, nullInt(actorID)); err != nil {
		m.lo.Error("error inserting blacklist log", "contact_id", contactID, "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing blacklist add", "contact_id", contactID, "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	m.lo.Info("contact added to blacklist", "contact_id", contactID, "actor_id", actorID)
	return nil
}

// Remove clears the blacklist flag of a contact and appends an operation log
// entry. It is idempotent: removing a contact that is not blacklisted simply
// records the operation.
func (m *Manager) Remove(contactID, actorID int) error {
	if contactID <= 0 {
		return envelope.NewError(envelope.InputError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	tx, err := m.db.BeginTxx(context.Background(), &sql.TxOptions{})
	if err != nil {
		m.lo.Error("error starting transaction for blacklist remove", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer tx.Rollback()

	if _, err := tx.Stmtx(m.q.DeleteBlacklist).Exec(contactID); err != nil {
		m.lo.Error("error deleting contact blacklist", "contact_id", contactID, "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if _, err := tx.Stmtx(m.q.InsertBlacklistLog).Exec(contactID, models.ActionRemove, "", nullInt(actorID)); err != nil {
		m.lo.Error("error inserting blacklist log", "contact_id", contactID, "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing blacklist remove", "contact_id", contactID, "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	m.lo.Info("contact removed from blacklist", "contact_id", contactID, "actor_id", actorID)
	return nil
}

// GetCustomers returns the paginated customer list along with the total row
// count and the number of currently blacklisted contacts (the tab badge).
func (m *Manager) GetCustomers(f CustomerFilters) ([]models.Customer, int, int, error) {
	page, pageSize := normalizePagination(f.Page, f.PageSize)
	offset := (page - 1) * pageSize
	filter := f.Filter
	if filter != FilterBlocked {
		filter = FilterAll
	}

	query := baseQuery(m.q.GetBlacklistCustomers) +
		" ORDER BY " + sortClause(customerSortColumns, f.OrderBy, "b.created_at", f.Order) +
		fmt.Sprintf(" LIMIT %d OFFSET %d", pageSize, offset)

	var rows []models.Customer
	if err := m.db.Select(&rows, query, filter, f.From, f.To, strings.TrimSpace(f.Search)); err != nil {
		m.lo.Error("error fetching blacklist customers", "error", err)
		return nil, 0, 0, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	total := 0
	if len(rows) > 0 {
		total = rows[0].Total
	}

	blockedCount := 0
	if err := m.q.CountBlacklisted.Get(&blockedCount); err != nil {
		m.lo.Error("error counting blacklisted contacts", "error", err)
		return nil, 0, 0, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return rows, total, blockedCount, nil
}

// GetLogs returns the paginated blacklist operation log along with the total
// row count.
func (m *Manager) GetLogs(f LogFilters) ([]models.Log, int, error) {
	page, pageSize := normalizePagination(f.Page, f.PageSize)
	offset := (page - 1) * pageSize
	action := f.Action
	if action != models.ActionAdd && action != models.ActionRemove {
		action = ""
	}

	query := baseQuery(m.q.GetBlacklistLogs) +
		" ORDER BY " + sortClause(logSortColumns, f.OrderBy, "l.created_at", f.Order) +
		fmt.Sprintf(" LIMIT %d OFFSET %d", pageSize, offset)

	var rows []models.Log
	if err := m.db.Select(&rows, query, action, f.From, f.To, strings.TrimSpace(f.Search)); err != nil {
		m.lo.Error("error fetching blacklist logs", "error", err)
		return nil, 0, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	total := 0
	if len(rows) > 0 {
		total = rows[0].Total
	}
	return rows, total, nil
}

// baseQuery strips a trailing statement terminator and whitespace so the
// manager can safely append ORDER BY / LIMIT / OFFSET clauses.
func baseQuery(q string) string {
	return strings.TrimRight(strings.TrimSpace(q), ";")
}

// sortClause resolves a whitelisted sort column and direction into a safe
// ORDER BY fragment. Unknown values fall back to the given default column.
func sortClause(allowed map[string]string, orderBy, defaultColumn, order string) string {
	column, ok := allowed[orderBy]
	if !ok {
		column = defaultColumn
	}
	dir := "DESC"
	if strings.EqualFold(order, "asc") {
		dir = "ASC"
	}
	return fmt.Sprintf("%s %s NULLS LAST", column, dir)
}

func normalizePagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize
}

// nullInt turns a zero user reference into NULL so the FK column stays valid.
func nullInt(v int) any {
	if v <= 0 {
		return nil
	}
	return v
}
