package main

import (
	"strconv"
	"strings"

	"github.com/abhinavxd/libredesk/internal/cannedresponse/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// personalCannedRequest is the request body used by the per-agent create /
// update endpoints. Only `Content` is exposed because `sort_order` is
// managed by the server to keep the ordering deterministic per user.
type personalCannedRequest struct {
	Content string `json:"content"`
}

// pinPathValues parses the `source` and `response_id` path values used by
// the pin / unpin endpoints. It centralises validation so the two handlers
// stay symmetric and report the same error messages.
func pinPathValues(r *fastglue.Request) (models.PinSource, int, error) {
	app := r.Context.(*App)
	rawSource := r.RequestCtx.UserValue("source").(string)
	idStr := r.RequestCtx.UserValue("id").(string)
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		return "", 0, envelope.NewError(envelope.InputError, app.i18n.T("validation.invalidCannedResponseID"), nil)
	}
	source := models.PinSource(rawSource)
	if source != models.PinSourceGlobal && source != models.PinSourcePersonal {
		return "", 0, envelope.NewError(envelope.InputError, app.i18n.T("validation.invalidValue"), nil)
	}
	return source, id, nil
}

// handleGetAgentCannedResponses returns the aggregate canned-response view
// for the requesting agent: the global category tree, the agent's own
// personal canned responses, and the agent's pin list with the resolved
// content baked in. The frontend uses this single payload to render the
// entire "常用语" sidebar tab.
func handleGetAgentCannedResponses(r *fastglue.Request) error {
	app := r.Context.(*App)
	auser := r.RequestCtx.UserValue("user").(amodels.User)
	view, err := app.cannedResponse.GetAgentView(auser.ID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(view)
}

// handleCreatePersonalCannedResponse appends a personal canned response to
// the requesting agent's own list. The content is trimmed and length-checked
// the same way as the admin-facing endpoint to keep the validation
// behaviour consistent.
func handleCreatePersonalCannedResponse(r *fastglue.Request) error {
	app := r.Context.(*App)
	auser := r.RequestCtx.UserValue("user").(amodels.User)
	req := personalCannedRequest{}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}
	req.Content = strings.TrimSpace(req.Content)
	if err := validateCannedResponseContent(app, req.Content); err != nil {
		return sendErrorEnvelope(r, err)
	}
	resp, err := app.cannedResponse.CreatePersonal(auser.ID, req.Content)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(resp)
}

// handleUpdatePersonalCannedResponse updates one of the requesting agent's
// own personal canned responses. The Manager enforces the `user_id` scope
// so a wrong id is reported as not found instead of leaking ownership.
func handleUpdatePersonalCannedResponse(r *fastglue.Request) error {
	app := r.Context.(*App)
	auser := r.RequestCtx.UserValue("user").(amodels.User)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.invalidCannedResponseID"), nil, envelope.InputError)
	}
	req := personalCannedRequest{}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}
	req.Content = strings.TrimSpace(req.Content)
	if err := validateCannedResponseContent(app, req.Content); err != nil {
		return sendErrorEnvelope(r, err)
	}
	resp, err := app.cannedResponse.UpdatePersonal(auser.ID, id, req.Content, 0)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(resp)
}

// handleDeletePersonalCannedResponse removes one of the requesting agent's
// own personal canned responses. Pin rows for the same personal response
// are cleaned up by FK cascade.
func handleDeletePersonalCannedResponse(r *fastglue.Request) error {
	app := r.Context.(*App)
	auser := r.RequestCtx.UserValue("user").(amodels.User)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.invalidCannedResponseID"), nil, envelope.InputError)
	}
	if err := app.cannedResponse.DeletePersonal(auser.ID, id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(struct{}{})
}

// handlePinCannedResponse pins a canned response (global or personal) to
// the requesting agent's "置顶常用语" list. Idempotent: re-pinning the same
// response is a no-op rather than an error.
func handlePinCannedResponse(r *fastglue.Request) error {
	app := r.Context.(*App)
	auser := r.RequestCtx.UserValue("user").(amodels.User)
	source, id, err := pinPathValues(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err := app.cannedResponse.Pin(auser.ID, source, id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(struct{}{})
}

// handleUnpinCannedResponse removes a canned response from the requesting
// agent's pin list. Idempotent.
func handleUnpinCannedResponse(r *fastglue.Request) error {
	app := r.Context.(*App)
	auser := r.RequestCtx.UserValue("user").(amodels.User)
	source, id, err := pinPathValues(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if err := app.cannedResponse.Unpin(auser.ID, source, id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(struct{}{})
}
