// Package keywordmatch implements the "关键字匹配" (keyword match) feature: a
// global list of rules matched against visitor and agent messages. A matching
// rule either alerts agents only (nothing is written to the conversation) or
// sends an automatic reply that both the agent and the visitor can see.
package keywordmatch

import (
	"database/sql"
	"embed"
	"regexp"
	"strings"
	"sync"
	"time"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/keywordmatch/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

// rulesCacheTTL is how long the in-process copy of the enabled rules is reused
// before it is reloaded from the DB. Writes invalidate it immediately, so this
// only bounds staleness across multiple app instances.
const rulesCacheTTL = 30 * time.Second

// ConversationService defines the conversation operations required by the
// keyword match module. It is implemented by the conversation manager.
type ConversationService interface {
	// SendAutoReply sends a message on behalf of the system user. The silent
	// flag controls whether the new-message websocket broadcast is skipped.
	SendAutoReply(inboxID, contactID int, conversationUUID, content string, metaMap map[string]any, silent bool) (cmodels.Message, error)
	// AuthorizedConnectedAgentIDs returns the IDs of the online agents allowed
	// to read the given assignment.
	AuthorizedConnectedAgentIDs(assignedUserID, assignedTeamID null.Int) []int
	// BroadcastToAgents pushes an arbitrary websocket event to the given agents.
	// An empty list broadcasts to every online agent.
	BroadcastToAgents(userIDs []int, eventType string, data map[string]any)
}

var (
	//go:embed queries.sql
	efs embed.FS
)

// Manager holds the keyword match rules and runs the matching engine.
type Manager struct {
	q    queries
	db   *sqlx.DB
	lo   *logf.Logger
	i18n *i18n.I18n
	conv ConversationService

	mu       sync.RWMutex
	rules    []models.KeywordMatchRule
	cachedAt time.Time

	reCache sync.Map
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB   *sqlx.DB
	Lo   *logf.Logger
	I18n *i18n.I18n
}

// queries contains prepared SQL queries.
type queries struct {
	GetAllRules     *sqlx.Stmt `query:"get-all-rules"`
	GetEnabledRules *sqlx.Stmt `query:"get-enabled-rules"`
	GetRule         *sqlx.Stmt `query:"get-rule"`
	InsertRule      *sqlx.Stmt `query:"insert-rule"`
	UpdateRule      *sqlx.Stmt `query:"update-rule"`
	DeleteRule      *sqlx.Stmt `query:"delete-rule"`
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

// SetConversationService wires the conversation service after construction to
// avoid an import cycle between the keywordmatch and conversation packages.
func (m *Manager) SetConversationService(conv ConversationService) {
	m.conv = conv
}

// GetAll returns every rule ordered by sort order.
func (m *Manager) GetAll() ([]models.KeywordMatchRule, error) {
	rules := make([]models.KeywordMatchRule, 0)
	if err := m.q.GetAllRules.Select(&rules); err != nil {
		m.lo.Error("error fetching keyword match rules", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return rules, nil
}

// Get returns a rule by ID.
func (m *Manager) Get(id int) (models.KeywordMatchRule, error) {
	var rule models.KeywordMatchRule
	if err := m.q.GetRule.Get(&rule, id); err != nil {
		if err == sql.ErrNoRows {
			return rule, envelope.NewError(envelope.InputError, m.i18n.T("validation.notFoundKeywordMatchRule"), nil)
		}
		m.lo.Error("error fetching keyword match rule", "id", id, "error", err)
		return rule, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return rule, nil
}

// Create creates a new rule.
func (m *Manager) Create(name, source, matchType, pattern, action, replyContent string, caseSensitive, searchFirst, enabled bool, sortOrder int) (models.KeywordMatchRule, error) {
	var rule models.KeywordMatchRule
	if err := m.validate(name, source, matchType, pattern, action); err != nil {
		return rule, err
	}
	if err := m.q.InsertRule.Get(&rule, name, source, matchType, pattern, caseSensitive, searchFirst, action, replyContent, enabled, sortOrder); err != nil {
		m.lo.Error("error inserting keyword match rule", "error", err)
		return rule, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	m.invalidateCache()
	return rule, nil
}

// Update updates an existing rule.
func (m *Manager) Update(id int, name, source, matchType, pattern, action, replyContent string, caseSensitive, searchFirst, enabled bool, sortOrder int) (models.KeywordMatchRule, error) {
	var rule models.KeywordMatchRule
	if err := m.validate(name, source, matchType, pattern, action); err != nil {
		return rule, err
	}
	if err := m.q.UpdateRule.Get(&rule, id, name, source, matchType, pattern, caseSensitive, searchFirst, action, replyContent, enabled, sortOrder); err != nil {
		if err == sql.ErrNoRows {
			return rule, envelope.NewError(envelope.InputError, m.i18n.T("validation.notFoundKeywordMatchRule"), nil)
		}
		m.lo.Error("error updating keyword match rule", "id", id, "error", err)
		return rule, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	m.invalidateCache()
	return rule, nil
}

// Delete deletes a rule.
func (m *Manager) Delete(id int) error {
	if _, err := m.q.DeleteRule.Exec(id); err != nil {
		m.lo.Error("error deleting keyword match rule", "id", id, "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	m.invalidateCache()
	return nil
}

// enabledRules returns the enabled rules, served from a short-lived in-process
// cache so message matching does not hit the DB on every message.
func (m *Manager) enabledRules() []models.KeywordMatchRule {
	m.mu.RLock()
	if m.rules != nil && time.Since(m.cachedAt) < rulesCacheTTL {
		rules := m.rules
		m.mu.RUnlock()
		return rules
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()
	// Re-check under the write lock: another goroutine may have refreshed.
	if m.rules != nil && time.Since(m.cachedAt) < rulesCacheTTL {
		return m.rules
	}
	rules := make([]models.KeywordMatchRule, 0)
	if err := m.q.GetEnabledRules.Select(&rules); err != nil {
		m.lo.Error("error fetching enabled keyword match rules", "error", err)
		// Fall back to the previous snapshot when available so a transient DB
		// error does not silently disable every rule.
		if m.rules == nil {
			return nil
		}
		return m.rules
	}
	m.rules = rules
	m.cachedAt = time.Now()
	return m.rules
}

func (m *Manager) invalidateCache() {
	m.mu.Lock()
	m.rules = nil
	m.cachedAt = time.Time{}
	m.mu.Unlock()
}

// validate checks the rule fields and compiles regex patterns up-front so an
// invalid pattern can never reach the matching engine.
func (m *Manager) validate(name, source, matchType, pattern, action string) error {
	if strings.TrimSpace(name) == "" {
		return envelope.NewError(envelope.InputError, m.i18n.Ts("globals.messages.empty", "name", m.i18n.T("keywordMatch.name")), nil)
	}
	if strings.TrimSpace(pattern) == "" {
		return envelope.NewError(envelope.InputError, m.i18n.Ts("globals.messages.empty", "name", m.i18n.T("keywordMatch.pattern")), nil)
	}
	if !models.IsValidSource(source) {
		return envelope.NewError(envelope.InputError, m.i18n.T("validation.invalidKeywordMatchSource"), nil)
	}
	if !models.IsValidMatchType(matchType) {
		return envelope.NewError(envelope.InputError, m.i18n.T("validation.invalidKeywordMatchType"), nil)
	}
	if !models.IsValidAction(action) {
		return envelope.NewError(envelope.InputError, m.i18n.T("validation.invalidKeywordMatchAction"), nil)
	}
	if matchType == models.MatchTypeRegex {
		if _, err := compileRegex(pattern, true); err != nil {
			return envelope.NewError(envelope.InputError, m.i18n.T("validation.invalidKeywordMatchPattern"), nil)
		}
	}
	return nil
}

// compileRegex compiles the pattern, optionally case-insensitively, reusing a
// process-wide cache of compiled expressions.
func compileRegex(pattern string, caseSensitive bool) (*regexp.Regexp, error) {
	key := pattern
	if caseSensitive {
		key = "cs:" + pattern
	}
	if cached, ok := regexCache.Load(key); ok {
		return cached.(*regexp.Regexp), nil
	}
	expr := pattern
	if !caseSensitive {
		expr = "(?i)" + pattern
	}
	re, err := regexp.Compile(expr)
	if err != nil {
		return nil, err
	}
	regexCache.Store(key, re)
	return re, nil
}

// regexCache is shared by every Manager instance; compiled expressions are
// immutable and safe for concurrent use.
var regexCache sync.Map
