// Package quickreply handles the management of automatic replies (quick replies)
// configured per inbox. It provides CRUD operations for the inbox quick reply
// config, topics, and questions.
package quickreply

import (
	"database/sql"
	"embed"
	"slices"
	"strings"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/quickreply/models"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/go-i18n"
	"github.com/lib/pq"
	"github.com/zerodha/logf"
)

// ConversationService defines the conversation operations required by the
// quick reply module to send automatic replies and read/update conversation
// metadata. It is implemented by the conversation manager.
type ConversationService interface {
	// SendAutoReply sends a message on behalf of the system user.
	SendAutoReply(inboxID, contactID int, conversationUUID, content string, metaMap map[string]any) (cmodels.Message, error)
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
}

var (
	//go:embed queries.sql
	efs embed.FS
)

type Manager struct {
	q    queries
	lo   *logf.Logger
	i18n *i18n.I18n
	conv ConversationService
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
		lo:   opts.Lo,
		i18n: opts.I18n,
		conv: opts.Conv,
	}, nil
}

// SetConversationService wires the conversation service after construction to
// avoid an import cycle between the quickreply and conversation packages.
func (m *Manager) SetConversationService(conv ConversationService) {
	m.conv = conv
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
func (m *Manager) UpsertConfig(inboxID int, welcomeMessage, transferKeyword, queueReply, assignedReply, closedReply string, enabled bool) (models.InboxQuickReplyConfig, error) {
	var cfg models.InboxQuickReplyConfig
	if err := m.q.UpsertConfig.Get(&cfg, inboxID, welcomeMessage, transferKeyword, queueReply, assignedReply, closedReply, enabled); err != nil {
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
	if _, err := m.conv.SendAutoReply(conversation.InboxID, conversation.ContactID, conversation.UUID, content, metaMap); err != nil {
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
				msg, err := m.conv.SendAutoReply(conversation.InboxID, conversation.ContactID, conversation.UUID, question.Answer, nil)
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
		} else if _, err := m.conv.SendAutoReply(conversation.InboxID, conversation.ContactID, conversation.UUID, cfg.AssignedReply, nil); err != nil {
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

	_, err = m.conv.SendAutoReply(conversation.InboxID, conversation.ContactID, conversation.UUID, cfg.ClosedReply, nil)
	return err
}

// handleTransferRequest sends the queue reply with the current queue position
// and marks the conversation as having requested a human agent.
func (m *Manager) handleTransferRequest(conversation cmodels.Conversation, cfg models.InboxQuickReplyConfig) error {
	count, err := m.conv.CountOpenUnassignedConversations()
	if err != nil {
		return err
	}

	metaMap := map[string]any{"type": cmodels.MessageMetaTypeQueueInfo, "count": count}
	if _, err := m.conv.SendAutoReply(conversation.InboxID, conversation.ContactID, conversation.UUID, cfg.QueueReply, metaMap); err != nil {
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

	// Refresh queue info for all waiting conversations so the freshly created
	// queue entry shows up immediately on other waiting widgets too.
	if err := m.conv.RefreshWaitingQueueInfo(); err != nil {
		m.lo.Error("error refreshing waiting queue info", "error", err)
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
	msg, err := m.conv.SendAutoReply(conversation.InboxID, conversation.ContactID, conversation.UUID, hintMsg, metaMap)
	return msg, err
}
