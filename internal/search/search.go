// Package search provides search functionality.
package search

import (
	"embed"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	models "github.com/abhinavxd/libredesk/internal/search/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/lib/pq"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

// Manager is the search manager
type Manager struct {
	q    queries
	lo   *logf.Logger
	i18n *i18n.I18n
}

// Opts contains the options for creating a new search manager
type Opts struct {
	DB   *sqlx.DB
	Lo   *logf.Logger
	I18n *i18n.I18n
}

// queries contains all the prepared queries
type queries struct {
	SearchConversationsByRefNum       *sqlx.Stmt `query:"search-conversations-by-reference-number"`
	SearchConversationsByContactEmail *sqlx.Stmt `query:"search-conversations-by-contact-email"`
	SearchMessages                    *sqlx.Stmt `query:"search-messages"`
	SearchContacts                    *sqlx.Stmt `query:"search-contacts"`
}

// New creates a new search manager
func New(opts Opts) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{q: q, lo: opts.Lo, i18n: opts.I18n}, nil
}

// filterArgs converts Filters into positional arguments in the order expected
// by the SQL queries: ($2 = start, $3 = end, $4 = inbox ids, $5 = agent id).
// nil values are returned for unset filters so the SQL `IS NULL` skip-pattern
// applies and the existing queries remain backward compatible.
func filterArgs(f models.Filters) (startAt, endAt interface{}, inboxIDs interface{}, agentID int) {
	if !f.StartAt.IsZero() {
		startAt = f.StartAt
	}
	if !f.EndAt.IsZero() {
		endAt = f.EndAt
	}
	if len(f.InboxIDs) > 0 {
		inboxIDs = pq.Array(f.InboxIDs)
	}
	if f.AgentID > 0 {
		agentID = f.AgentID
	}
	return
}

func reverseConversationResults(s []models.ConversationResult) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

func reverseMessageResults(s []models.MessageResult) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

// Conversations searches conversations based on the query and the optional filters.
func (s *Manager) Conversations(query string, f models.Filters) ([]models.ConversationResult, error) {
	startAt, endAt, inboxIDs, agentID := filterArgs(f)

	var refNumResults = make([]models.ConversationResult, 0)
	if err := s.q.SearchConversationsByRefNum.Select(&refNumResults, query, startAt, endAt, inboxIDs, agentID); err != nil {
		s.lo.Error("error searching conversations", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, s.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	var emailResults = make([]models.ConversationResult, 0)
	if err := s.q.SearchConversationsByContactEmail.Select(&emailResults, query, startAt, endAt, inboxIDs, agentID); err != nil {
		s.lo.Error("error searching conversations", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, s.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	results := append(refNumResults, emailResults...)
	if f.Sort == "oldest" {
		reverseConversationResults(results)
	}
	return results, nil
}

// Messages searches messages based on the query and the optional filters.
func (s *Manager) Messages(query string, f models.Filters) ([]models.MessageResult, error) {
	startAt, endAt, inboxIDs, agentID := filterArgs(f)

	var results = make([]models.MessageResult, 0)
	if err := s.q.SearchMessages.Select(&results, query, startAt, endAt, inboxIDs, agentID); err != nil {
		s.lo.Error("error searching messages", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, s.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if f.Sort == "oldest" {
		reverseMessageResults(results)
	}
	return results, nil
}

// Contacts searches contacts based on the query (no additional filters).
func (s *Manager) Contacts(query string) ([]models.ContactResult, error) {
	var results = make([]models.ContactResult, 0)
	if err := s.q.SearchContacts.Select(&results, query); err != nil {
		s.lo.Error("error searching contacts", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, s.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return results, nil
}
