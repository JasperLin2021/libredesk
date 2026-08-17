package main

import (
	"strconv"
	"strings"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// maxQuickReplyFieldLength limits the length of text fields.
const maxQuickReplyFieldLength = 10000

// quickReplyConfigRequest is the request body for updating the quick reply
// configuration of an inbox.
type quickReplyConfigRequest struct {
	WelcomeMessage  string `json:"welcome_message"`
	TransferKeyword string `json:"transfer_keyword"`
	QueueReply      string `json:"queue_reply"`
	AssignedReply   string `json:"assigned_reply"`
	ClosedReply     string `json:"closed_reply"`
	Enabled         bool   `json:"enabled"`
}

// handleGetQuickReplyConfig returns the quick reply config of an inbox.
func handleGetQuickReplyConfig(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		id, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.invalidInboxID"), nil, envelope.InputError)
	}
	cfg, err := app.quickReply.GetConfig(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(cfg)
}

// handleUpdateQuickReplyConfig creates or updates the quick reply config of an inbox.
func handleUpdateQuickReplyConfig(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		req     = quickReplyConfigRequest{}
		id, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.invalidInboxID"), nil, envelope.InputError)
	}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}
	req.WelcomeMessage = strings.TrimSpace(req.WelcomeMessage)
	req.TransferKeyword = strings.TrimSpace(req.TransferKeyword)
	req.QueueReply = strings.TrimSpace(req.QueueReply)
	req.AssignedReply = strings.TrimSpace(req.AssignedReply)
	req.ClosedReply = strings.TrimSpace(req.ClosedReply)
	if req.TransferKeyword == "" {
		req.TransferKeyword = "我要转人工"
	}
	if len(req.WelcomeMessage) > maxQuickReplyFieldLength || len(req.TransferKeyword) > 500 ||
		len(req.QueueReply) > maxQuickReplyFieldLength || len(req.AssignedReply) > maxQuickReplyFieldLength ||
		len(req.ClosedReply) > maxQuickReplyFieldLength {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.inputTooLong"), nil, envelope.InputError)
	}
	cfg, err := app.quickReply.UpsertConfig(id, req.WelcomeMessage, req.TransferKeyword, req.QueueReply, req.AssignedReply, req.ClosedReply, req.Enabled)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(cfg)
}

// handleGetQuickReplyTopics returns the topics (with their questions) of an inbox.
func handleGetQuickReplyTopics(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		id, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.invalidInboxID"), nil, envelope.InputError)
	}
	topics, err := app.quickReply.GetTopicsWithQuestions(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(topics)
}

// handleCreateQuickReplyTopic creates a new topic under an inbox.
func handleCreateQuickReplyTopic(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		req     = struct {
			Name        string   `json:"name"`
			Names       []string `json:"names"`
			HintMessage string   `json:"hint_message"`
			SortOrder   int      `json:"sort_order"`
		}{}
		id, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.invalidInboxID"), nil, envelope.InputError)
	}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`name`"), nil, envelope.InputError)
	}
	if len(req.Name) > 500 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.inputTooLong"), nil, envelope.InputError)
	}
	// Build the names array: primary name first, then any aliases.
	names := buildNames(req.Name, req.Names)
	if err := validateNames(id, names, 0, app); err != nil {
		return err
	}
	topic, err := app.quickReply.CreateTopic(id, req.Name, names, req.HintMessage, req.SortOrder)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(topic)
}

// handleUpdateQuickReplyTopic updates a topic.
func handleUpdateQuickReplyTopic(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		req     = struct {
			Name        string   `json:"name"`
			Names       []string `json:"names"`
			HintMessage string   `json:"hint_message"`
			SortOrder   int      `json:"sort_order"`
		}{}
		id, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.notFoundQuickReplyTopic"), nil, envelope.InputError)
	}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`name`"), nil, envelope.InputError)
	}
	if len(req.Name) > 500 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.inputTooLong"), nil, envelope.InputError)
	}
	// Build the names array: primary name first, then any aliases.
	names := buildNames(req.Name, req.Names)
	// Fetch the existing topic to get its inbox ID for uniqueness validation.
	existing, err := app.quickReply.GetTopic(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err := validateNames(int(existing.InboxID), names, int64(id), app); err != nil {
		return err
	}
	topic, err := app.quickReply.UpdateTopic(id, req.Name, names, req.HintMessage, req.SortOrder)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(topic)
}

// buildNames returns a deduplicated slice of names with the primary name as the
// first element. Empty/whitespace-only aliases are discarded.
func buildNames(primary string, aliases []string) []string {
	seen := map[string]bool{primary: true}
	names := []string{primary}
	for _, a := range aliases {
		a = strings.TrimSpace(a)
		if a == "" || seen[a] {
			continue
		}
		seen[a] = true
		names = append(names, a)
	}
	return names
}

// validateNames checks that every name is non-empty, within length limits, and
// unique within the inbox.
func validateNames(inboxID int, names []string, excludeID int64, app *App) error {
	for _, n := range names {
		if len(n) > 500 {
			return envelope.NewError(envelope.InputError, app.i18n.T("validation.inputTooLong"), nil)
		}
		exists, err := app.quickReply.TopicNameExists(inboxID, n, excludeID)
		if err != nil {
			return err
		}
		if exists {
			return envelope.NewError(envelope.ConflictError, app.i18n.T("quickReply.topicNameExists"), nil)
		}
	}
	return nil
}

// handleDeleteQuickReplyTopic deletes a topic and its questions.
func handleDeleteQuickReplyTopic(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		id, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.notFoundQuickReplyTopic"), nil, envelope.InputError)
	}
	if err := app.quickReply.DeleteTopic(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// handleCreateQuickReplyQuestion creates a new question under a topic.
func handleCreateQuickReplyQuestion(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		req     = struct {
			Question  string `json:"question"`
			Answer    string `json:"answer"`
			SortOrder int    `json:"sort_order"`
		}{}
		id, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.notFoundQuickReplyTopic"), nil, envelope.InputError)
	}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}
	req.Question = strings.TrimSpace(req.Question)
	if req.Question == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`question`"), nil, envelope.InputError)
	}
	if len(req.Question) > 500 || len(req.Answer) > maxQuickReplyFieldLength {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.inputTooLong"), nil, envelope.InputError)
	}
	question, err := app.quickReply.CreateQuestion(id, req.Question, strings.TrimSpace(req.Answer), req.SortOrder)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(question)
}

// handleUpdateQuickReplyQuestion updates a question.
func handleUpdateQuickReplyQuestion(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		req     = struct {
			Question  string `json:"question"`
			Answer    string `json:"answer"`
			SortOrder int    `json:"sort_order"`
		}{}
		id, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.notFoundQuickReplyQuestion"), nil, envelope.InputError)
	}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}
	req.Question = strings.TrimSpace(req.Question)
	if req.Question == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`question`"), nil, envelope.InputError)
	}
	if len(req.Question) > 500 || len(req.Answer) > maxQuickReplyFieldLength {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.inputTooLong"), nil, envelope.InputError)
	}
	question, err := app.quickReply.UpdateQuestion(id, req.Question, strings.TrimSpace(req.Answer), req.SortOrder)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(question)
}

// handleDeleteQuickReplyQuestion deletes a question.
func handleDeleteQuickReplyQuestion(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		id, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.notFoundQuickReplyQuestion"), nil, envelope.InputError)
	}
	if err := app.quickReply.DeleteQuestion(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}
