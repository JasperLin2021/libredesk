package main

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/abhinavxd/libredesk/internal/cannedresponse"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// Limits on how much data a single import payload may contain, to keep the
// request bounded.
const (
	maxCannedImportCategories = 2000
	maxCannedImportResponses  = 20000
)

// cannedCategoryRequest is the request body for creating or updating a category.
type cannedCategoryRequest struct {
	Name      string `json:"name"`
	ParentID  *int64 `json:"parent_id"`
	SortOrder int    `json:"sort_order"`
}

// cannedResponseRequest is the request body for creating or updating a canned
// response.
type cannedResponseRequest struct {
	CategoryID int64  `json:"category_id"`
	Content    string `json:"content"`
	SortOrder  int    `json:"sort_order"`
}

// cannedImportRequest is the request body for bulk importing a category tree.
type cannedImportRequest struct {
	Categories []cannedresponse.ImportItem `json:"categories"`
}

// handleGetCannedResponses returns the full category tree with the canned
// responses attached to each category.
func handleGetCannedResponses(r *fastglue.Request) error {
	app := r.Context.(*App)
	tree, err := app.cannedResponse.GetTree()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(tree)
}

// handleCreateCannedCategory creates a new category.
func handleCreateCannedCategory(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = cannedCategoryRequest{}
	)
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}
	req.Name = strings.TrimSpace(req.Name)
	if err := validateCannedCategoryName(app, req.Name); err != nil {
		return sendErrorEnvelope(r, err)
	}
	category, err := app.cannedResponse.CreateCategory(req.ParentID, req.Name)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(category)
}

// handleUpdateCannedCategory renames, re-parents and/or reorders a category.
func handleUpdateCannedCategory(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		req     = cannedCategoryRequest{}
		id, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.invalidCannedCategoryID"), nil, envelope.InputError)
	}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}
	req.Name = strings.TrimSpace(req.Name)
	if err := validateCannedCategoryName(app, req.Name); err != nil {
		return sendErrorEnvelope(r, err)
	}
	category, err := app.cannedResponse.UpdateCategory(id, req.ParentID, req.Name, req.SortOrder)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(category)
}

// handleDeleteCannedCategory deletes a category (and, by cascade, its
// sub-categories and responses).
func handleDeleteCannedCategory(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		id, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.invalidCannedCategoryID"), nil, envelope.InputError)
	}
	if err := app.cannedResponse.DeleteCategory(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(struct{}{})
}

// handleCreateCannedResponse creates a new canned response.
func handleCreateCannedResponse(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = cannedResponseRequest{}
	)
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}
	if req.CategoryID == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.invalidCannedCategoryID"), nil, envelope.InputError)
	}
	req.Content = strings.TrimSpace(req.Content)
	if err := validateCannedResponseContent(app, req.Content); err != nil {
		return sendErrorEnvelope(r, err)
	}
	response, err := app.cannedResponse.CreateResponse(int(req.CategoryID), req.Content)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(response)
}

// handleUpdateCannedResponse updates the content and/or order of a canned
// response.
func handleUpdateCannedResponse(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		req     = cannedResponseRequest{}
		id, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.invalidCannedResponseID"), nil, envelope.InputError)
	}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}
	req.Content = strings.TrimSpace(req.Content)
	if err := validateCannedResponseContent(app, req.Content); err != nil {
		return sendErrorEnvelope(r, err)
	}
	response, err := app.cannedResponse.UpdateResponse(id, req.Content, req.SortOrder)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(response)
}

// handleDeleteCannedResponse deletes a single canned response.
func handleDeleteCannedResponse(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		id, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.invalidCannedResponseID"), nil, envelope.InputError)
	}
	if err := app.cannedResponse.DeleteResponse(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(struct{}{})
}

// handleDeleteAllCannedResponses deletes every category and canned response.
func handleDeleteAllCannedResponses(r *fastglue.Request) error {
	app := r.Context.(*App)
	if err := app.cannedResponse.DeleteAll(); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(struct{}{})
}

// handleImportCannedResponses bulk-imports a category tree parsed from an
// uploaded spreadsheet. Existing categories and responses are kept; matching
// categories are reused and new responses are appended.
func handleImportCannedResponses(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = cannedImportRequest{}
	)
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}
	if len(req.Categories) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.cannedImportEmpty"), nil, envelope.InputError)
	}

	var categoryCount, responseCount int
	if err := validateCannedImportItems(app, req.Categories, &categoryCount, &responseCount); err != nil {
		return sendErrorEnvelope(r, err)
	}
	if categoryCount > maxCannedImportCategories || responseCount > maxCannedImportResponses {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.cannedImportTooLarge"), nil, envelope.InputError)
	}

	stats, err := app.cannedResponse.Import(req.Categories)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(stats)
}

// validateCannedCategoryName ensures a category name is present and within the
// allowed length.
func validateCannedCategoryName(app *App, name string) error {
	if name == "" {
		return envelope.NewError(envelope.InputError, app.i18n.T("validation.cannedCategoryNameRequired"), nil)
	}
	if utf8.RuneCountInString(name) > cannedresponse.MaxCategoryNameLength {
		return envelope.NewError(envelope.InputError, app.i18n.T("validation.cannedCategoryNameTooLong"), nil)
	}
	return nil
}

// validateCannedResponseContent ensures a canned response has content within
// the allowed length.
func validateCannedResponseContent(app *App, content string) error {
	if content == "" {
		return envelope.NewError(envelope.InputError, app.i18n.T("validation.cannedResponseContentRequired"), nil)
	}
	if utf8.RuneCountInString(content) > cannedresponse.MaxResponseLength {
		return envelope.NewError(envelope.InputError, app.i18n.T("validation.cannedResponseContentTooLong"), nil)
	}
	return nil
}

// validateCannedImportItems recursively validates an import payload and counts
// its categories and responses.
func validateCannedImportItems(app *App, items []cannedresponse.ImportItem, categoryCount, responseCount *int) error {
	for _, it := range items {
		name := strings.TrimSpace(it.Name)
		if name == "" {
			continue
		}
		if utf8.RuneCountInString(name) > cannedresponse.MaxCategoryNameLength {
			return envelope.NewError(envelope.InputError, app.i18n.T("validation.cannedCategoryNameTooLong"), nil)
		}
		*categoryCount++
		for _, content := range it.Responses {
			content = strings.TrimSpace(content)
			if content == "" {
				continue
			}
			if utf8.RuneCountInString(content) > cannedresponse.MaxResponseLength {
				return envelope.NewError(envelope.InputError, app.i18n.T("validation.cannedResponseContentTooLong"), nil)
			}
			*responseCount++
		}
		if len(it.Children) > 0 {
			if err := validateCannedImportItems(app, it.Children, categoryCount, responseCount); err != nil {
				return err
			}
		}
	}
	return nil
}
