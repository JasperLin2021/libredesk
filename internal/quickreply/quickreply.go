// Package quickreply handles the management of automatic replies (quick replies)
// configured per inbox. It provides CRUD operations for the inbox quick reply
// config, topics, and questions.
package quickreply

import (
	"database/sql"
	"embed"
	"encoding/json"
	"slices"
	"strings"
	"time"

	bhmodels "github.com/abhinavxd/libredesk/internal/business_hours/models"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/abhinavxd/libredesk/internal/quickreply/models"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
	"github.com/knadh/go-i18n"
	"github.com/lib/pq"
	"github.com/zerodha/logf"
)

// ConversationService defines the conversation operations required by the
// quick reply module to send automatic replies and read/update conversation
// metadata. It is implemented by the conversation manager.
type ConversationService interface {
	// SendAutoReply sends a message on behalf of the system user. The silent
	// flag controls whether the new-message websocket broadcast is skipped,
	// letting the quick reply module decide whether bot messages should
	// surface the conversation in the agent open list.
	SendAutoReply(inboxID, contactID int, conversationUUID, content string, metaMap map[string]any, silent bool) (cmodels.Message, error)
	// GetConversationMeta returns the conversation's meta as a map.
	GetConversationMeta(conversationUUID string) (map[string]any, error)
	// UpdateConversationMeta merges the given keys into the conversation's meta.
	UpdateConversationMeta(conversationUUID string, meta map[string]any) error
	// DeleteConversationMetaKey removes a single key from the conversation's meta.
	// Unlike UpdateConversationMeta (JSONB union, cannot delete keys), this can
	// actually remove an existing key.
	DeleteConversationMetaKey(conversationUUID, key string) error
	// CountOpenUnassignedConversations returns the number of open conversations
	// that are not assigned to any agent or team (used for queue position).
	CountOpenUnassignedConversations() (int, error)
	// SendReplyAsUser sends a message on behalf of the given agent user.
	SendReplyAsUser(userID, inboxID, contactID int, conversationUUID, content string, metaMap map[string]any) (cmodels.Message, error)
	// RefreshWaitingQueueInfo recomputes the queue count for every conversation
	// waiting for a human agent and pushes the updated count to their widgets.
	RefreshWaitingQueueInfo() error
	// CloseConversation closes the given conversation on behalf of the system
	// user. Closing a conversation triggers the configured closed reply.
	CloseConversation(uuid string) error
	// BroadcastNewConversation pushes a new-conversation websocket event so the
	// conversation surfaces in the agent open list. Used by the transfer
	// request flow to notify agents only after the visitor explicitly asks
	// for a human.
	BroadcastNewConversationByUUID(conversationUUID string) error
}

// InboxService provides read access to the inbox record. It is implemented by
// the inbox manager and only exposes the handoff configuration lookup needed to
// decide whether a transfer to a human agent is currently allowed.
type InboxService interface {
	// GetDBRecord returns the inbox record from the DB by numeric ID or UUID.
	GetDBRecord(identifier any) (imodels.Inbox, error)
}

// BusinessHoursService provides read access to a single business hours record.
// It is implemented by the business hours manager.
type BusinessHoursService interface {
	// Get retrieves business hours by ID.
	Get(id int) (bhmodels.BusinessHours, error)
	// IsWithinBusinessHours reports whether `now` falls within the given
	// business hours schedule, resolved in the supplied time zone.
	IsWithinBusinessHours(bh bhmodels.BusinessHours, tz string, now time.Time) (bool, error)
}

// SettingsService provides read access to the app settings, keyed by prefix. It
// is implemented by the settings manager.
type SettingsService interface {
	// GetByPrefix retrieves all settings starting with the given prefix.
	GetByPrefix(prefix string) (types.JSONText, error)
}

var (
	//go:embed queries.sql
	efs embed.FS
)

type Manager struct {
	q             queries
	db            *sqlx.DB
	lo            *logf.Logger
	i18n          *i18n.I18n
	conv          ConversationService
	inbox         InboxService
	businessHours BusinessHoursService
	settings      SettingsService
	// now returns the current time. It defaults to time.Now and is overridden
	// in tests so the handoff time window can be exercised with a fixed clock.
	now func() time.Time
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB   *sqlx.DB
	Lo   *logf.Logger
	I18n *i18n.I18n
	Conv ConversationService
}

// queries contains prepared SQL queries.
type queries struct {
	GetConfig      *sqlx.Stmt `query:"get-config"`
	UpsertConfig   *sqlx.Stmt `query:"upsert-config"`
	DeleteConfig   *sqlx.Stmt `query:"delete-config"`
	GetTopics      *sqlx.Stmt `query:"get-topics"`
	GetTopic       *sqlx.Stmt `query:"get-topic"`
	GetTopicByName *sqlx.Stmt `query:"get-topic-by-name"`
	InsertTopic    *sqlx.Stmt `query:"insert-topic"`
	UpdateTopic    *sqlx.Stmt `query:"update-topic"`
	DeleteTopic    *sqlx.Stmt `query:"delete-topic"`
	GetQuestions   *sqlx.Stmt `query:"get-questions-by-topic"`
	GetQuestion    *sqlx.Stmt `query:"get-question"`
	InsertQuestion *sqlx.Stmt `query:"insert-question"`
	UpdateQuestion *sqlx.Stmt `query:"update-question"`
	DeleteQuestion *sqlx.Stmt `query:"delete-question"`
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
		conv: opts.Conv,
		now:  time.Now,
	}, nil
}

// SetConversationService wires the conversation service after construction to
// avoid an import cycle between the quickreply and conversation packages.
func (m *Manager) SetConversationService(conv ConversationService) {
	m.conv = conv
}

// SetInboxService wires the inbox service after construction to avoid an import
// cycle between the quickreply and inbox packages. It is required for the
// handoff time window check; without it handoff is unrestricted.
func (m *Manager) SetInboxService(inbox InboxService) {
	m.inbox = inbox
}

// SetBusinessHoursService wires the business hours service after construction.
// It is required for the handoff time window check; without it handoff is
// unrestricted.
func (m *Manager) SetBusinessHoursService(businessHours BusinessHoursService) {
	m.businessHours = businessHours
}

// SetSettingsService wires the settings service after construction. It supplies
// the app time zone used to resolve the handoff time window; without it the
// window falls back to UTC.
func (m *Manager) SetSettingsService(settings SettingsService) {
	m.settings = settings
}

// GetConfig returns the quick reply config for the given inbox. If no config
// exists, a zero-value config and a nil error are returned so callers can
// treat the inbox as "quick reply disabled".
func (m *Manager) GetConfig(inboxID int) (models.InboxQuickReplyConfig, error) {
	var cfg models.InboxQuickReplyConfig
	if err := m.q.GetConfig.Get(&cfg, inboxID); err != nil {
		if err == sql.ErrNoRows {
			return cfg, nil
		}
		m.lo.Error("error fetching quick reply config", "error", err)
		return cfg, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return cfg, nil
}

// UpsertConfig creates or updates the quick reply config for the given inbox.
func (m *Manager) UpsertConfig(inboxID int, welcomeMessage, transferKeyword, queueReply, transferRestrictedMessage, assignedReply, closedReply, noReplyTimeoutReply string, enabled bool) (models.InboxQuickReplyConfig, error) {
	var cfg models.InboxQuickReplyConfig
	if err := m.q.UpsertConfig.Get(&cfg, inboxID, welcomeMessage, transferKeyword, queueReply, transferRestrictedMessage, assignedReply, closedReply, noReplyTimeoutReply, enabled); err != nil {
		if dbutil.IsForeignKeyError(err) {
			return cfg, envelope.NewError(envelope.InputError, m.i18n.T("validation.notFoundInbox"), nil)
		}
		m.lo.Error("error upserting quick reply config", "error", err)
		return cfg, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return cfg, nil
}

// DeleteConfig deletes the quick reply config for the given inbox.
func (m *Manager) DeleteConfig(inboxID int) error {
	if _, err := m.q.DeleteConfig.Exec(inboxID); err != nil {
		m.lo.Error("error deleting quick reply config", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

// GetTopics returns the topics configured for the given inbox.
func (m *Manager) GetTopics(inboxID int) ([]models.QuickReplyTopic, error) {
	var topics = make([]models.QuickReplyTopic, 0)
	if err := m.q.GetTopics.Select(&topics, inboxID); err != nil {
		m.lo.Error("error fetching quick reply topics", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return topics, nil
}

// GetTopicsWithQuestions returns the topics configured for the given inbox,
// with each topic's questions loaded.
func (m *Manager) GetTopicsWithQuestions(inboxID int) ([]models.QuickReplyTopic, error) {
	topics, err := m.GetTopics(inboxID)
	if err != nil {
		return nil, err
	}
	for i := range topics {
		questions, err := m.GetQuestions(int(topics[i].ID))
		if err != nil {
			return nil, err
		}
		topics[i].Questions = questions
	}
	return topics, nil
}

// GetTopic returns a topic by ID.
func (m *Manager) GetTopic(id int) (models.QuickReplyTopic, error) {
	var topic models.QuickReplyTopic
	if err := m.q.GetTopic.Get(&topic, id); err != nil {
		if err == sql.ErrNoRows {
			return topic, envelope.NewError(envelope.InputError, m.i18n.T("validation.notFoundQuickReplyTopic"), nil)
		}
		m.lo.Error("error fetching quick reply topic", "error", err)
		return topic, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return topic, nil
}

// CreateTopic creates a new topic under the given inbox.
// names contains all display names (aliases) for the topic; the first element
// is treated as the primary name.
func (m *Manager) CreateTopic(inboxID int, name string, names []string, hintMessage string, sortOrder int) (models.QuickReplyTopic, error) {
	var topic models.QuickReplyTopic
	if err := m.q.InsertTopic.Get(&topic, inboxID, name, pq.Array(names), hintMessage, sortOrder); err != nil {
		if dbutil.IsUniqueViolationError(err) {
			return topic, envelope.NewError(envelope.ConflictError, m.i18n.T("errors.alreadyExistsQuickReplyTopic"), nil)
		}
		if dbutil.IsForeignKeyError(err) {
			return topic, envelope.NewError(envelope.InputError, m.i18n.T("validation.notFoundInbox"), nil)
		}
		m.lo.Error("error inserting quick reply topic", "error", err)
		return topic, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return topic, nil
}

// UpdateTopic updates a topic by ID.
// names contains all display names (aliases) for the topic; the first element
// is treated as the primary name.
func (m *Manager) UpdateTopic(id int, name string, names []string, hintMessage string, sortOrder int) (models.QuickReplyTopic, error) {
	var topic models.QuickReplyTopic
	if err := m.q.UpdateTopic.Get(&topic, id, name, pq.Array(names), hintMessage, sortOrder); err != nil {
		if err == sql.ErrNoRows {
			return topic, envelope.NewError(envelope.InputError, m.i18n.T("validation.notFoundQuickReplyTopic"), nil)
		}
		if dbutil.IsUniqueViolationError(err) {
			return topic, envelope.NewError(envelope.ConflictError, m.i18n.T("errors.alreadyExistsQuickReplyTopic"), nil)
		}
		m.lo.Error("error updating quick reply topic", "error", err)
		return topic, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return topic, nil
}

// GetTopicByName returns the topic that has the given name (or alias) within
// the specified inbox.
func (m *Manager) GetTopicByName(inboxID int, name string) (models.QuickReplyTopic, error) {
	var topic models.QuickReplyTopic
	if err := m.q.GetTopicByName.Get(&topic, inboxID, name); err != nil {
		if err == sql.ErrNoRows {
			return topic, envelope.NewError(envelope.InputError, m.i18n.T("validation.notFoundQuickReplyTopic"), nil)
		}
		m.lo.Error("error fetching quick reply topic by name", "error", err)
		return topic, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return topic, nil
}

// TopicNameExists checks whether any topic in the given inbox already uses the
// specified name (including aliases). excludeID can be set to skip a specific
// topic (useful when updating).
func (m *Manager) TopicNameExists(inboxID int, name string, excludeID int64) (bool, error) {
	topic, err := m.GetTopicByName(inboxID, name)
	if err != nil {
		// If not found, no conflict.
		return false, nil
	}
	if excludeID > 0 && topic.ID == excludeID {
		return false, nil
	}
	return true, nil
}

// DeleteTopic deletes a topic and its questions by ID.
func (m *Manager) DeleteTopic(id int) error {
	if _, err := m.q.DeleteTopic.Exec(id); err != nil {
		m.lo.Error("error deleting quick reply topic", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

// GetQuestions returns the questions under the given topic.
func (m *Manager) GetQuestions(topicID int) ([]models.QuickReplyQuestion, error) {
	var questions = make([]models.QuickReplyQuestion, 0)
	if err := m.q.GetQuestions.Select(&questions, topicID); err != nil {
		m.lo.Error("error fetching quick reply questions", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return questions, nil
}

// GetQuestion returns a question by ID.
func (m *Manager) GetQuestion(id int) (models.QuickReplyQuestion, error) {
	var question models.QuickReplyQuestion
	if err := m.q.GetQuestion.Get(&question, id); err != nil {
		if err == sql.ErrNoRows {
			return question, envelope.NewError(envelope.InputError, m.i18n.T("validation.notFoundQuickReplyQuestion"), nil)
		}
		m.lo.Error("error fetching quick reply question", "error", err)
		return question, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return question, nil
}

// CreateQuestion creates a new question under the given topic.
func (m *Manager) CreateQuestion(topicID int, question, answer string, sortOrder int) (models.QuickReplyQuestion, error) {
	var q models.QuickReplyQuestion
	if err := m.q.InsertQuestion.Get(&q, topicID, question, answer, sortOrder); err != nil {
		if dbutil.IsForeignKeyError(err) {
			return q, envelope.NewError(envelope.InputError, m.i18n.T("validation.notFoundQuickReplyTopic"), nil)
		}
		m.lo.Error("error inserting quick reply question", "error", err)
		return q, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return q, nil
}

// UpdateQuestion updates a question by ID.
func (m *Manager) UpdateQuestion(id int, question, answer string, sortOrder int) (models.QuickReplyQuestion, error) {
	var q models.QuickReplyQuestion
	if err := m.q.UpdateQuestion.Get(&q, id, question, answer, sortOrder); err != nil {
		if err == sql.ErrNoRows {
			return q, envelope.NewError(envelope.InputError, m.i18n.T("validation.notFoundQuickReplyQuestion"), nil)
		}
		m.lo.Error("error updating quick reply question", "error", err)
		return q, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return q, nil
}

// DeleteQuestion deletes a question by ID.
func (m *Manager) DeleteQuestion(id int) error {
	if _, err := m.q.DeleteQuestion.Exec(id); err != nil {
		m.lo.Error("error deleting quick reply question", "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return nil
}

// ImportTopicItem represents a single topic to import together with its
// questions. It mirrors the payload of the bulk import endpoint.
type ImportTopicItem struct {
	Name        string               `json:"name"`
	Names       []string             `json:"names"`
	HintMessage string               `json:"hint_message"`
	Questions   []ImportQuestionItem `json:"questions"`
}

// ImportQuestionItem represents a single question (with its answer) to import.
type ImportQuestionItem struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// ImportStats contains the number of topics and questions created or updated
// during a bulk import.
type ImportStats struct {
	CreatedTopics    int `json:"created_topics"`
	UpdatedTopics    int `json:"updated_topics"`
	CreatedQuestions int `json:"created_questions"`
	UpdatedQuestions int `json:"updated_questions"`
}

// ImportTopics bulk-imports topics (with their questions) into the given inbox
// using an append-merge strategy:
//   - A topic is matched by any of its names (primary name or aliases). When it
//     matches an existing topic, that topic is updated: the names are merged
//     (deduplicated, primary name first), the hint message is replaced and the
//     questions are upserted by question text (existing questions are updated,
//     new ones are created).
//   - A topic that matches nothing is created together with all its questions.
//   - No existing data is ever deleted.
//
// The whole import runs inside a single database transaction. On any error the
// transaction is rolled back so a failed import never leaves partial writes.
func (m *Manager) ImportTopics(inboxID int, topics []ImportTopicItem) (ImportStats, error) {
	var stats ImportStats
	tx, err := m.db.Beginx()
	if err != nil {
		m.lo.Error("error starting quick reply import transaction", "error", err)
		return stats, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer tx.Rollback() //nolint:errcheck

	// New topics are appended after the existing ones.
	existingTopics, err := m.GetTopics(inboxID)
	if err != nil {
		return stats, err
	}
	nextTopicSort := len(existingTopics)

	for _, item := range topics {
		names := mergeTopicNames([]string{item.Name}, item.Names)
		if len(names) == 0 {
			continue
		}

		existing, err := m.resolveImportTopic(tx, inboxID, names)
		if err != nil {
			return stats, err
		}

		if existing.ID > 0 {
			// Update the existing topic, merging names, hint message and questions.
			mergedNames := mergeTopicNames(existing.Names, names)
			if _, err := tx.Stmtx(m.q.UpdateTopic).Exec(existing.ID, mergedNames[0], pq.Array(mergedNames), item.HintMessage, existing.SortOrder); err != nil {
				if dbutil.IsUniqueViolationError(err) {
					return stats, envelope.NewError(envelope.ConflictError, m.i18n.T("errors.alreadyExistsQuickReplyTopic"), nil)
				}
				m.lo.Error("error updating quick reply topic during import", "error", err)
				return stats, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
			}
			stats.UpdatedTopics++

			var existingQuestions []models.QuickReplyQuestion
			if err := tx.Stmtx(m.q.GetQuestions).Select(&existingQuestions, existing.ID); err != nil {
				m.lo.Error("error fetching quick reply questions during import", "error", err)
				return stats, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
			}
			nextQuestionSort := len(existingQuestions)
			matched := make(map[int64]bool, len(existingQuestions))
			for _, qi := range item.Questions {
				target, found := findImportQuestion(existingQuestions, qi.Question, matched)
				if found {
					if _, err := tx.Stmtx(m.q.UpdateQuestion).Exec(target.ID, qi.Question, qi.Answer, target.SortOrder); err != nil {
						m.lo.Error("error updating quick reply question during import", "error", err)
						return stats, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
					}
					matched[target.ID] = true
					stats.UpdatedQuestions++
				} else {
					if _, err := tx.Stmtx(m.q.InsertQuestion).Exec(existing.ID, qi.Question, qi.Answer, nextQuestionSort); err != nil {
						m.lo.Error("error inserting quick reply question during import", "error", err)
						return stats, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
					}
					nextQuestionSort++
					stats.CreatedQuestions++
				}
			}
		} else {
			// Create a new topic together with its questions.
			var topic models.QuickReplyTopic
			if err := tx.Stmtx(m.q.InsertTopic).Get(&topic, inboxID, names[0], pq.Array(names), item.HintMessage, nextTopicSort); err != nil {
				if dbutil.IsUniqueViolationError(err) {
					return stats, envelope.NewError(envelope.ConflictError, m.i18n.T("errors.alreadyExistsQuickReplyTopic"), nil)
				}
				if dbutil.IsForeignKeyError(err) {
					return stats, envelope.NewError(envelope.InputError, m.i18n.T("validation.notFoundInbox"), nil)
				}
				m.lo.Error("error inserting quick reply topic during import", "error", err)
				return stats, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
			}
			stats.CreatedTopics++
			nextTopicSort++

			for idx, qi := range item.Questions {
				if _, err := tx.Stmtx(m.q.InsertQuestion).Exec(topic.ID, qi.Question, qi.Answer, idx); err != nil {
					m.lo.Error("error inserting quick reply question during import", "error", err)
					return stats, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
				}
				stats.CreatedQuestions++
			}
		}
	}

	if err := tx.Commit(); err != nil {
		m.lo.Error("error committing quick reply import transaction", "error", err)
		return stats, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	return stats, nil
}

// resolveImportTopic returns the existing topic that the given names resolve to
// within the inbox. Every name must resolve to the same topic; when different
// names match different topics a conflict error is returned. A zero topic is
// returned when no name matches anything.
func (m *Manager) resolveImportTopic(tx *sqlx.Tx, inboxID int, names []string) (models.QuickReplyTopic, error) {
	var matched models.QuickReplyTopic
	for _, name := range names {
		var topic models.QuickReplyTopic
		if err := tx.Stmtx(m.q.GetTopicByName).Get(&topic, inboxID, name); err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			m.lo.Error("error fetching quick reply topic by name during import", "error", err)
			return models.QuickReplyTopic{}, envelope.NewError(envelope.GeneralError, m.i18n.T("globals.messages.somethingWentWrong"), nil)
		}
		if matched.ID == 0 {
			matched = topic
			continue
		}
		if topic.ID != matched.ID {
			return models.QuickReplyTopic{}, envelope.NewError(envelope.ConflictError, m.i18n.T("quickReply.topicNameExists"), nil)
		}
	}
	return matched, nil
}

// mergeTopicNames merges a primary name list with an extra name list into a
// single deduplicated slice, preserving the order of the primary list and
// appending unseen names from the extra list. Empty names are dropped.
func mergeTopicNames(primary, extra []string) []string {
	names := make([]string, 0, len(primary)+len(extra))
	seen := make(map[string]bool, len(primary)+len(extra))
	add := func(n string) {
		n = strings.TrimSpace(n)
		if n == "" || seen[n] {
			return
		}
		seen[n] = true
		names = append(names, n)
	}
	for _, n := range primary {
		add(n)
	}
	for _, n := range extra {
		add(n)
	}
	return names
}

// findImportQuestion returns the first existing question whose text matches the
// imported question. Already matched questions are skipped so every existing
// question is updated at most once.
func findImportQuestion(questions []models.QuickReplyQuestion, question string, matched map[int64]bool) (models.QuickReplyQuestion, bool) {
	for _, q := range questions {
		if matched[q.ID] {
			continue
		}
		if q.Question == question {
			return q, true
		}
	}
	return models.QuickReplyQuestion{}, false
}

// SendWelcomeReply sends the configured welcome message with quick reply topic
// cards to a freshly created conversation. It is a no-op when quick reply is
// disabled for the inbox or the welcome message is empty.
func (m *Manager) SendWelcomeReply(conversation cmodels.Conversation) error {
	m.lo.Info("quick reply: SendWelcomeReply called", "conversation_uuid", conversation.UUID, "inbox_id", conversation.InboxID)
	if m.conv == nil {
		m.lo.Warn("quick reply: conversation service is nil, skipping welcome reply", "conversation_uuid", conversation.UUID)
		return nil
	}
	cfg, err := m.GetConfig(conversation.InboxID)
	if err != nil {
		m.lo.Error("quick reply: error getting config", "inbox_id", conversation.InboxID, "error", err)
		return nil
	}
	m.lo.Info("quick reply: config loaded", "inbox_id", conversation.InboxID, "conversation_uuid", conversation.UUID, "enabled", cfg.Enabled, "welcome_message_length", len(cfg.WelcomeMessage))
	if !cfg.Enabled {
		m.lo.Debug("quick reply: config not enabled for inbox", "inbox_id", conversation.InboxID, "conversation_uuid", conversation.UUID)
		return nil
	}
	// Build topic items (optional — welcome message is sent even without topics).
	items := make([]map[string]string, 0)
	topics, err := m.GetTopics(conversation.InboxID)
	if err == nil {
		for _, topic := range topics {
			items = append(items, map[string]string{"label": topic.Name, "value": topic.Name})
		}
	}

	hasWelcomeMessage := strings.TrimSpace(cfg.WelcomeMessage) != ""
	hasTopics := len(items) > 0

	if !hasWelcomeMessage && !hasTopics {
		m.lo.Debug("quick reply: no welcome message and no topics for inbox", "inbox_id", conversation.InboxID, "conversation_uuid", conversation.UUID)
		return nil
	}

	m.lo.Info("quick reply: sending welcome message", "conversation_uuid", conversation.UUID, "inbox_id", conversation.InboxID, "contact_id", conversation.ContactID, "has_welcome_message", hasWelcomeMessage, "has_topics", hasTopics)

	// Use empty content when no welcome message is configured
	// but topics are available, so the topic cards are still delivered.
	content := cfg.WelcomeMessage
	if !hasWelcomeMessage {
		content = ""
	}

	metaMap := map[string]any{
		"type":             cmodels.MessageMetaTypeBotQuickReply,
		"items":            items,
		"transfer_keyword": cfg.TransferKeyword,
	}
	// Welcome message is a bot auto-reply, keep the visitor's conversation
	// hidden from the agent open list until they request a human.
	if _, err := m.conv.SendAutoReply(conversation.InboxID, conversation.ContactID, conversation.UUID, content, metaMap, true); err != nil {
		m.lo.Error("quick reply: error sending welcome auto reply", "conversation_uuid", conversation.UUID, "error", err)
		return err
	}
	m.lo.Info("quick reply: welcome message sent successfully", "conversation_uuid", conversation.UUID)
	return nil
}

// HandleIncomingMessage processes a visitor's message and sends an automatic
// reply when the message matches the configured transfer keyword, a topic, or
// a question. It is a no-op when quick reply is disabled. Once the visitor has
// requested a transfer to a human agent or the conversation is already assigned
// to an agent, all auto replies are suppressed — including topic / question card
// clicks — so the visitor interacts with the human agent only. It returns the
// list of bot messages created.
func (m *Manager) HandleIncomingMessage(conversation cmodels.Conversation, content string) ([]cmodels.Message, error) {
	if m.conv == nil {
		return nil, nil
	}
	cfg, err := m.GetConfig(conversation.InboxID)
	if err != nil || !cfg.Enabled {
		return nil, nil
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return nil, nil
	}

	// Once the visitor requested a transfer to a human agent or the conversation
	// is already assigned to an agent, stop all auto replies (including topic /
	// question card clicks) so the visitor interacts with the human agent only.
	if conversation.AssignedUserID.Valid || conversation.AssignedTeamID.Valid {
		return nil, nil
	}
	meta, err := m.conv.GetConversationMeta(conversation.UUID)
	if err != nil {
		return nil, nil
	}
	if requested, _ := meta[cmodels.ConversationMetaBotHumanRequested].(bool); requested {
		return nil, nil
	}

	// Transfer to human agent.
	if cfg.TransferKeyword != "" && content == cfg.TransferKeyword {
		return nil, m.handleTransferRequest(conversation, cfg)
	}

	// Match against configured topics and questions. Topic / question cards clicked
	// by the visitor always get a reply, so the auto-reply flow keeps working until
	// the visitor asks to be transferred or a human agent takes over.
	topics, err := m.GetTopicsWithQuestions(conversation.InboxID)
	if err != nil {
		return nil, nil
	}
	for _, topic := range topics {
		// Exact topic match against any of the topic's names (aliases).
		if topicMatchesName(topic, content) {
			msg, err := m.sendTopicQuestions(conversation, topic)
			if err != nil {
				return nil, err
			}
			if msg.ID > 0 {
				return []cmodels.Message{msg}, nil
			}
			return nil, nil
		}
	}
	// Exact question match: send the answer.
	for _, topic := range topics {
		for _, question := range topic.Questions {
			if question.Question == content {
				msg, err := m.conv.SendAutoReply(conversation.InboxID, conversation.ContactID, conversation.UUID, question.Answer, nil, true)
				if err != nil {
					return nil, err
				}
				return []cmodels.Message{msg}, nil
			}
		}
	}

	// Free-form text that does not match any topic or question never triggers an
	// automatic reply.
	return nil, nil
}

// HandleUserAssigned sends the "you are now connected to a human agent" reply
// when a conversation that requested a human transfer gets assigned to an agent.
// The transfer marker is cleared afterwards.
func (m *Manager) HandleUserAssigned(conversation cmodels.Conversation) error {
	if m.conv == nil {
		return nil
	}

	meta, err := m.conv.GetConversationMeta(conversation.UUID)
	if err != nil {
		return nil
	}
	requested, _ := meta[cmodels.ConversationMetaBotHumanRequested].(bool)
	if !requested {
		return nil
	}

	// Send the "you are now connected to a human agent" reply if configured.
	// When the conversation is assigned to a specific agent the reply is sent
	// on behalf of that agent so the widget shows their avatar and name;
	// otherwise fall back to the system user.
	cfg, err := m.GetConfig(conversation.InboxID)
	if err == nil && cfg.Enabled && strings.TrimSpace(cfg.AssignedReply) != "" {
		if conversation.AssignedUserID.Valid {
			if _, err := m.conv.SendReplyAsUser(conversation.AssignedUserID.Int, conversation.InboxID, conversation.ContactID, conversation.UUID, cfg.AssignedReply, nil); err != nil {
				m.lo.Error("error sending quick reply assigned message as agent", "conversation_uuid", conversation.UUID, "assigned_user_id", conversation.AssignedUserID.Int, "error", err)
			}
		} else if _, err := m.conv.SendAutoReply(conversation.InboxID, conversation.ContactID, conversation.UUID, cfg.AssignedReply, nil, true); err != nil {
			m.lo.Error("error sending quick reply assigned message", "conversation_uuid", conversation.UUID, "error", err)
		}
	}

	// Clear the transfer marker. Once a human agent takes over, the conversation
	// is guarded by the assignment check in HandleIncomingMessage, so the marker
	// must be removed even when no assigned reply is configured — otherwise it
	// lingers and silently kills auto-replies after the conversation is closed
	// and reopened (or unassigned) later.
	if err := m.conv.DeleteConversationMetaKey(conversation.UUID, cmodels.ConversationMetaBotHumanRequested); err != nil {
		m.lo.Error("error clearing quick reply transfer marker", "conversation_uuid", conversation.UUID, "error", err)
	}
	// Remove the persisted queue position; the widget hides its queue footer
	// once the conversation is assigned.
	if err := m.conv.DeleteConversationMetaKey(conversation.UUID, cmodels.ConversationMetaQueueInfo); err != nil {
		m.lo.Error("error clearing quick reply queue info", "conversation_uuid", conversation.UUID, "error", err)
	}
	return nil
}

// HandleConversationClosed sends the configured closed reply when a conversation
// is moved to the closed status.
func (m *Manager) HandleConversationClosed(conversation cmodels.Conversation) error {
	if m.conv == nil {
		return nil
	}
	cfg, err := m.GetConfig(conversation.InboxID)
	if err != nil || !cfg.Enabled || strings.TrimSpace(cfg.ClosedReply) == "" {
		return nil
	}

	_, err = m.conv.SendAutoReply(conversation.InboxID, conversation.ContactID, conversation.UUID, cfg.ClosedReply, nil, true)
	return err
}

// HandleNoReplyTimeout handles a conversation in which the visitor has not
// replied within the configured no-reply timeout: it sends the configured
// "no reply timeout" message on behalf of the assigned agent and then closes
// the conversation (which triggers the configured closed reply through the
// regular status-change pipeline). Each conversation is handled at most once,
// guarded by the no_reply_timeout_sent conversation meta marker.
func (m *Manager) HandleNoReplyTimeout(conversation cmodels.Conversation) error {
	if m.conv == nil {
		return nil
	}

	// Only assigned-to-agent conversations are eligible.
	if !conversation.AssignedUserID.Valid {
		return nil
	}

	// Guard against duplicate handling (e.g. overlapping scanner cycles).
	meta, err := m.conv.GetConversationMeta(conversation.UUID)
	if err != nil {
		m.lo.Error("error fetching conversation meta for no reply timeout", "conversation_uuid", conversation.UUID, "error", err)
		return err
	}
	if sent, _ := meta[cmodels.ConversationMetaNoReplyTimeoutSent].(bool); sent {
		return nil
	}

	// Mark the conversation as handled before sending so that concurrent
	// scanner cycles never send the message twice.
	if err := m.conv.UpdateConversationMeta(conversation.UUID, map[string]any{cmodels.ConversationMetaNoReplyTimeoutSent: true}); err != nil {
		m.lo.Error("error marking no reply timeout as handled", "conversation_uuid", conversation.UUID, "error", err)
		return err
	}

	// Send the "no reply timeout" message on behalf of the assigned agent when
	// configured; otherwise skip straight to closing the conversation so the
	// closed reply still fires.
	if cfg, err := m.GetConfig(conversation.InboxID); err == nil && cfg.Enabled && strings.TrimSpace(cfg.NoReplyTimeoutReply) != "" {
		if _, err := m.conv.SendReplyAsUser(conversation.AssignedUserID.Int, conversation.InboxID, conversation.ContactID, conversation.UUID, cfg.NoReplyTimeoutReply, nil); err != nil {
			// A send failure must not block the conversation from being closed.
			m.lo.Error("error sending no reply timeout message as agent", "conversation_uuid", conversation.UUID, "assigned_user_id", conversation.AssignedUserID.Int, "error", err)
		}
	}

	// Close the conversation. This runs through the regular status-change
	// pipeline and triggers the configured closed reply afterwards.
	if err := m.conv.CloseConversation(conversation.UUID); err != nil {
		m.lo.Error("error closing conversation after no reply timeout", "conversation_uuid", conversation.UUID, "error", err)
		return err
	}
	return nil
}

// handleTransferRequest sends the queue reply with the current queue position
// and marks the conversation as having requested a human agent. It also
// surfaces the conversation in the agent open list, because the visitor has
// now explicitly asked to talk to a human.
//
// When the inbox restricts handoff to a time window ("转人工设置"), the transfer
// only proceeds inside that window. Outside it the configured restriction
// message is sent and the conversation stays in the bot flow.
func (m *Manager) handleTransferRequest(conversation cmodels.Conversation, cfg models.InboxQuickReplyConfig) error {
	// Gate the transfer on the inbox's handoff time window. Outside the allowed
	// window the visitor gets the restriction message and nothing else happens.
	if !m.handoffAllowedNow(conversation) {
		return m.sendRestrictedReply(conversation, cfg)
	}

	count, err := m.conv.CountOpenUnassignedConversations()
	if err != nil {
		return err
	}

	metaMap := map[string]any{"type": cmodels.MessageMetaTypeQueueInfo, "count": count}
	// Queue reply is a bot auto-reply, keep it silent so the broadcast is
	// driven solely by the explicit transfer event below.
	if _, err := m.conv.SendAutoReply(conversation.InboxID, conversation.ContactID, conversation.UUID, cfg.QueueReply, metaMap, true); err != nil {
		m.lo.Error("error sending quick reply queue message", "conversation_uuid", conversation.UUID, "error", err)
	}

	// Persist the queue position and mark the conversation as having requested
	// a transfer to a human agent. The queue_info meta drives the widget's
	// persistent queue footer until the conversation gets assigned.
	convMeta, err := m.conv.GetConversationMeta(conversation.UUID)
	if err == nil {
		convMeta[cmodels.ConversationMetaBotHumanRequested] = true
		convMeta[cmodels.ConversationMetaQueueInfo] = map[string]any{"count": count}
		if err := m.conv.UpdateConversationMeta(conversation.UUID, convMeta); err != nil {
			m.lo.Error("error marking quick reply transfer request", "conversation_uuid", conversation.UUID, "error", err)
		}
	}

	// The visitor just clicked "Talk to a human" – surface the conversation
	// in the agent open list. If the conversation was previously assigned
	// (e.g. an agent had finished a session and the visitor is asking again)
	// the broadcast is still useful to bring the conversation back into the
	// active queue.
	if err := m.conv.BroadcastNewConversationByUUID(conversation.UUID); err != nil {
		m.lo.Error("error broadcasting transfer request", "conversation_uuid", conversation.UUID, "error", err)
	}

	// Refresh queue info for all waiting conversations so the freshly created
	// queue entry shows up immediately on other waiting widgets too.
	if err := m.conv.RefreshWaitingQueueInfo(); err != nil {
		m.lo.Error("error refreshing waiting queue info", "error", err)
	}
	return nil
}

// handoffAllowedNow reports whether the visitor may be transferred to a human
// agent at the current time, based on the inbox's "转人工设置" (handoff time
// window) configuration.
//
// A NULL or zero `handoff_business_hours_id` means unrestricted, so the
// existing "always allow" behaviour is preserved. Any failure looking up the
// inbox, business hours, or settings is treated as "allowed" and logged so
// that a misconfigured schedule can never silently block visitors from
// reaching a human agent.
func (m *Manager) handoffAllowedNow(conversation cmodels.Conversation) bool {
	if m.inbox == nil || m.businessHours == nil {
		return true
	}
	inbox, err := m.inbox.GetDBRecord(int(conversation.InboxID))
	if err != nil {
		m.lo.Error("error fetching inbox for handoff time window check", "inbox_id", conversation.InboxID, "error", err)
		return true
	}
	if !inbox.HandoffBusinessHoursID.Valid || inbox.HandoffBusinessHoursID.Int <= 0 {
		return true
	}
	bh, err := m.businessHours.Get(int(inbox.HandoffBusinessHoursID.Int))
	if err != nil {
		m.lo.Error("error fetching business hours for handoff time window check", "business_hours_id", inbox.HandoffBusinessHoursID.Int, "error", err)
		return true
	}
	tz := m.appTimezone()
	inHours, err := m.businessHours.IsWithinBusinessHours(bh, tz, m.now())
	if err != nil {
		m.lo.Error("error evaluating handoff time window", "business_hours_id", inbox.HandoffBusinessHoursID.Int, "tz", tz, "error", err)
		return true
	}
	// `business` only allows handoff during working hours, `off_business` is
	// the inverse. `NormalizedHandoffHoursMode` defaults unknown values to
	// `business` so an old / corrupt row is always interpreted the safe way.
	if inbox.NormalizedHandoffHoursMode() == imodels.HandoffHoursModeOffBusiness {
		return !inHours
	}
	return inHours
}

// appTimezone returns the app-wide time zone from settings (key
// `app.timezone`). It falls back to UTC on any failure so the caller can use
// the result unconditionally.
func (m *Manager) appTimezone() string {
	const fallback = "UTC"
	if m.settings == nil {
		return fallback
	}
	raw, err := m.settings.GetByPrefix("app")
	if err != nil || len(raw) == 0 {
		return fallback
	}
	settings := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		m.lo.Error("error unmarshalling app settings for time zone", "error", err)
		return fallback
	}
	tz, _ := settings["app.timezone"].(string)
	if tz == "" {
		return fallback
	}
	return tz
}

// sendRestrictedReply sends the "transfer restricted" reply when the visitor
// asks for a human agent outside the inbox's allowed handoff time window. The
// conversation is intentionally left untouched: no bot_human_requested marker,
// no queue info, no broadcast. This way the visitor can ask again later in the
// allowed window and be queued normally.
func (m *Manager) sendRestrictedReply(conversation cmodels.Conversation, cfg models.InboxQuickReplyConfig) error {
	reply := strings.TrimSpace(cfg.TransferRestrictedMessage)
	if reply == "" {
		// Fall back to a default so the visitor never receives an empty
		// message. The i18n key is the same one surfaced in the admin form
		// description so admins see the same string visitors receive.
		reply = m.i18n.T("quickReply.transferRestricted")
	}
	if _, err := m.conv.SendAutoReply(conversation.InboxID, conversation.ContactID, conversation.UUID, reply, nil, true); err != nil {
		m.lo.Error("error sending quick reply transfer restricted message", "conversation_uuid", conversation.UUID, "error", err)
		return err
	}
	return nil
}

// topicMatchesName returns true if the given content matches any of the topic's
// names (primary name or aliases).
func topicMatchesName(topic models.QuickReplyTopic, content string) bool {
	return slices.Contains(topic.Names, content) || topic.Name == content
}

// sendTopicQuestions sends the hint message and question cards of the given topic.
func (m *Manager) sendTopicQuestions(conversation cmodels.Conversation, topic models.QuickReplyTopic) (cmodels.Message, error) {
	if len(topic.Questions) == 0 {
		return cmodels.Message{}, nil
	}
	items := make([]map[string]string, 0, len(topic.Questions))
	for _, question := range topic.Questions {
		items = append(items, map[string]string{"label": question.Question, "value": question.Question})
	}
	metaMap := map[string]any{"type": cmodels.MessageMetaTypeBotQuickReply, "items": items}
	// Use topic's hint message if set, otherwise fall back to i18n default.
	hintMsg := topic.HintMessage
	if strings.TrimSpace(hintMsg) == "" {
		hintMsg = m.i18n.T("quickReply.selectQuestion")
	}
	msg, err := m.conv.SendAutoReply(conversation.InboxID, conversation.ContactID, conversation.UUID, hintMsg, metaMap, true)
	return msg, err
}
