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
			Name      string `json:"name"`
			SortOrder int    `json:"sort_order"`
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
	topic, err := app.quickReply.CreateTopic(id, req.Name, req.SortOrder)
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
			Name      string `json:"name"`
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
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.Ts("globals.messages.empty", "name", "`name`"), nil, envelope.InputError)
	}
	if len(req.Name) > 500 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.inputTooLong"), nil, envelope.InputError)
	}
	topic, err := app.quickReply.UpdateTopic(id, req.Name, req.SortOrder)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(topic)
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
