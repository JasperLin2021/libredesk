package main

import (
	"strconv"
	"strings"

	"github.com/abhinavxd/libredesk/internal/envelope"
	kmodels "github.com/abhinavxd/libredesk/internal/keywordmatch/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

const (
	maxKeywordMatchNameLength    = 300
	maxKeywordMatchPatternLength = 2000
	maxKeywordMatchReplyLength   = 10000
)

// keywordMatchRuleRequest is the request body for creating / updating a keyword
// match rule (关键字匹配).
type keywordMatchRuleRequest struct {
	Name          string `json:"name"`
	Source        string `json:"source"`
	MatchType     string `json:"match_type"`
	Pattern       string `json:"pattern"`
	CaseSensitive bool   `json:"case_sensitive"`
	SearchFirst   bool   `json:"search_first"`
	Action        string `json:"action"`
	ReplyContent  string `json:"reply_content"`
	Enabled       *bool  `json:"enabled"`
	SortOrder     int    `json:"sort_order"`
}

// handleGetKeywordMatchRules returns every keyword match rule.
func handleGetKeywordMatchRules(r *fastglue.Request) error {
	var app = r.Context.(*App)
	rules, err := app.keywordMatch.GetAll()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(rules)
}

// handleGetKeywordMatchRule returns a single keyword match rule.
func handleGetKeywordMatchRule(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		id, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.invalidKeywordMatchRuleID"), nil, envelope.InputError)
	}
	rule, err := app.keywordMatch.Get(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(rule)
}

// handleCreateKeywordMatchRule creates a keyword match rule.
func handleCreateKeywordMatchRule(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = keywordMatchRuleRequest{}
	)
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}
	req, err := normalizeKeywordMatchRequest(app, req)
	if err != nil {
		return err
	}
	rule, err := app.keywordMatch.Create(req.Name, req.Source, req.MatchType, req.Pattern, req.Action, req.ReplyContent, req.CaseSensitive, req.SearchFirst, isEnabled(req.Enabled), req.SortOrder)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(rule)
}

// handleUpdateKeywordMatchRule updates a keyword match rule.
func handleUpdateKeywordMatchRule(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		req     = keywordMatchRuleRequest{}
		id, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.invalidKeywordMatchRuleID"), nil, envelope.InputError)
	}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}
	req, err = normalizeKeywordMatchRequest(app, req)
	if err != nil {
		return err
	}
	rule, err := app.keywordMatch.Update(id, req.Name, req.Source, req.MatchType, req.Pattern, req.Action, req.ReplyContent, req.CaseSensitive, req.SearchFirst, isEnabled(req.Enabled), req.SortOrder)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(rule)
}

// handleDeleteKeywordMatchRule deletes a keyword match rule.
func handleDeleteKeywordMatchRule(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		id, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.invalidKeywordMatchRuleID"), nil, envelope.InputError)
	}
	if err := app.keywordMatch.Delete(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(true)
}

// normalizeKeywordMatchRequest trims the text fields, applies the defaults for
// the optional enum fields and enforces the field length limits.
func normalizeKeywordMatchRequest(app *App, req keywordMatchRuleRequest) (keywordMatchRuleRequest, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.Source = strings.TrimSpace(req.Source)
	req.MatchType = strings.TrimSpace(req.MatchType)
	req.Pattern = strings.TrimSpace(req.Pattern)
	req.Action = strings.TrimSpace(req.Action)

	if req.Source == "" {
		req.Source = kmodels.SourceVisitor
	}
	if req.MatchType == "" {
		req.MatchType = kmodels.MatchTypeKeyword
	}
	if req.Action == "" {
		req.Action = kmodels.ActionAgentNotify
	}
	if len(req.Name) > maxKeywordMatchNameLength ||
		len(req.Pattern) > maxKeywordMatchPatternLength ||
		len(req.ReplyContent) > maxKeywordMatchReplyLength {
		return req, envelope.NewError(envelope.InputError, app.i18n.T("validation.inputTooLong"), nil)
	}
	return req, nil
}

// isEnabled reports whether a rule should be enabled. A missing flag defaults
// to enabled so rules created without the field are active right away.
func isEnabled(enabled *bool) bool {
	if enabled == nil {
		return true
	}
	return *enabled
}
