package main

import (
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/knowledgebase"
	"github.com/abhinavxd/libredesk/internal/knowledgebase/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// Maximum number of articles in a single bulk-import request, used to keep
// the request bounded and the validation error messages actionable.
const (
	maxKnowledgeImportRows = 5000
)

// knowledgeCategoryRequest is the request body for creating or updating a
// knowledge base category.
type knowledgeCategoryRequest struct {
	Name      string  `json:"name"`
	ParentID  *int64  `json:"parent_id"`
	Code      *string `json:"code"`
	SortOrder int     `json:"sort_order"`
}

// knowledgeArticleRequest is the request body for creating or updating a
// knowledge article.
type knowledgeArticleRequest struct {
	CategoryID  *int64                         `json:"category_id"`
	ExternalID  *string                        `json:"external_id"`
	Code        *string                        `json:"code"`
	Title       string                         `json:"title"`
	Author      *string                        `json:"author"`
	Content     string                         `json:"content"`
	Status      models.ArticleStatus           `json:"status"`
	SortOrder   int                            `json:"sort_order"`
	Attachments []models.AttachmentInput       `json:"attachments"`
}

// knowledgeImportRequest is the request body for the bulk importer.
type knowledgeImportRequest struct {
	Items []knowledgebase.ImportItem `json:"items"`
}

// handleGetKnowledgeCategories returns the full knowledge base category
// tree.
func handleGetKnowledgeCategories(r *fastglue.Request) error {
	app := r.Context.(*App)
	tree, err := app.knowledgeBase.GetTree()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(tree)
}

// handleCreateKnowledgeCategory creates a new knowledge base category.
func handleCreateKnowledgeCategory(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = knowledgeCategoryRequest{}
	)
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}
	req.Name = strings.TrimSpace(req.Name)
	if err := validateKnowledgeCategoryName(app, req.Name); err != nil {
		return sendErrorEnvelope(r, err)
	}
	if req.Code != nil {
		trimmed := strings.TrimSpace(*req.Code)
		if trimmed == "" {
			req.Code = nil
		} else {
			req.Code = &trimmed
		}
	}
	category, err := app.knowledgeBase.CreateCategory(req.ParentID, req.Code, req.Name)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(category)
}

// handleUpdateKnowledgeCategory renames, re-parents and/or reorders a
// knowledge base category.
func handleUpdateKnowledgeCategory(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		req     = knowledgeCategoryRequest{}
		id, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.invalidKnowledgeCategoryID"), nil, envelope.InputError)
	}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}
	req.Name = strings.TrimSpace(req.Name)
	if err := validateKnowledgeCategoryName(app, req.Name); err != nil {
		return sendErrorEnvelope(r, err)
	}
	if req.Code != nil {
		trimmed := strings.TrimSpace(*req.Code)
		if trimmed == "" {
			req.Code = nil
		} else {
			req.Code = &trimmed
		}
	}
	category, err := app.knowledgeBase.UpdateCategory(id, req.ParentID, req.Code, req.Name, req.SortOrder)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(category)
}

// handleDeleteKnowledgeCategory deletes a knowledge base category. The
// handler refuses to delete a category that still has articles attached
// (directly or under a sub-category) so callers can show a meaningful
// error in the UI before any data is lost.
func handleDeleteKnowledgeCategory(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		id, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.invalidKnowledgeCategoryID"), nil, envelope.InputError)
	}
	has, err := app.knowledgeBase.CategoryHasArticles(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if has {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.knowledgeCategoryNotEmpty"), nil, envelope.InputError)
	}
	if err := app.knowledgeBase.DeleteCategory(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(struct{}{})
}

// handleListKnowledgeArticles returns a paginated list of knowledge
// articles filtered by the request query string.
func handleListKnowledgeArticles(r *fastglue.Request) error {
	app := r.Context.(*App)
	filter, err := parseKnowledgeArticleFilter(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	res, err := app.knowledgeBase.List(filter)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(res)
}

// handleGetKnowledgeArticle returns a single knowledge article with its
// attachments.
func handleGetKnowledgeArticle(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		id, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.invalidKnowledgeArticleID"), nil, envelope.InputError)
	}
	article, err := app.knowledgeBase.GetArticle(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(article)
}

// handleCreateKnowledgeArticle creates a new knowledge article.
func handleCreateKnowledgeArticle(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = knowledgeArticleRequest{}
	)
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}
	in, err := buildKnowledgeArticleInput(&req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	article, err := app.knowledgeBase.CreateArticle(in)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(article)
}

// handleUpdateKnowledgeArticle updates an existing knowledge article and
// replaces its attachments.
func handleUpdateKnowledgeArticle(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		req     = knowledgeArticleRequest{}
		id, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.invalidKnowledgeArticleID"), nil, envelope.InputError)
	}
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}
	in, err := buildKnowledgeArticleInput(&req)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	article, err := app.knowledgeBase.UpdateArticle(id, knowledgebase.UpdateArticleInput{
		CategoryID:  in.CategoryID,
		ExternalID:  in.ExternalID,
		Code:        in.Code,
		Title:       in.Title,
		Author:      in.Author,
		Content:     in.Content,
		Status:      in.Status,
		SortOrder:   req.SortOrder,
		Attachments: in.Attachments,
	})
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(article)
}

// handleDeleteKnowledgeArticle deletes a knowledge article by id.
func handleDeleteKnowledgeArticle(r *fastglue.Request) error {
	var (
		app     = r.Context.(*App)
		id, err = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.invalidKnowledgeArticleID"), nil, envelope.InputError)
	}
	if err := app.knowledgeBase.DeleteArticle(id); err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(struct{}{})
}

// handleImportKnowledgeArticles bulk-imports a batch of knowledge
// articles from a parsed spreadsheet. The whole batch is processed in a
// single transaction; per-row errors are reported back without aborting
// the rest of the import.
func handleImportKnowledgeArticles(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		req = knowledgeImportRequest{}
	)
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), err.Error(), envelope.InputError)
	}
	if len(req.Items) == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.knowledgeImportEmpty"), nil, envelope.InputError)
	}
	if len(req.Items) > maxKnowledgeImportRows {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.knowledgeImportTooLarge"), nil, envelope.InputError)
	}
	for i := range req.Items {
		req.Items[i].CategoryCode = strings.TrimSpace(req.Items[i].CategoryCode)
		req.Items[i].CategoryName = strings.TrimSpace(req.Items[i].CategoryName)
		req.Items[i].ExternalID = strings.TrimSpace(req.Items[i].ExternalID)
		req.Items[i].ArticleCode = strings.TrimSpace(req.Items[i].ArticleCode)
		req.Items[i].Title = strings.TrimSpace(req.Items[i].Title)
		req.Items[i].Author = strings.TrimSpace(req.Items[i].Author)
	}
	stats, err := app.knowledgeBase.Import(req.Items)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(stats)
}

// handleExportKnowledgeArticles returns the full list of knowledge
// articles as a denormalised payload, ready to be formatted as a
// spreadsheet on the client side.
func handleExportKnowledgeArticles(r *fastglue.Request) error {
	app := r.Context.(*App)
	rows, err := app.knowledgeBase.ListAllForExport()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(rows)
}

// parseKnowledgeArticleFilter extracts a knowledgebase.ListFilter from the
// request's query string.
func parseKnowledgeArticleFilter(r *fastglue.Request) (knowledgebase.ListFilter, error) {
	var (
		f   knowledgebase.ListFilter
		err error
		app = r.Context.(*App)
	)
	args := r.RequestCtx.QueryArgs()
	if v := string(args.Peek("category_id")); v != "" {
		cid, perr := strconv.ParseInt(v, 10, 64)
		if perr != nil {
			return f, envelope.NewError(envelope.InputError, app.i18n.T("validation.invalidKnowledgeCategoryID"), nil)
		}
		f.CategoryID = &cid
	}
	if v := string(args.Peek("parent_category_id")); v != "" {
		pid, perr := strconv.ParseInt(v, 10, 64)
		if perr != nil {
			return f, envelope.NewError(envelope.InputError, app.i18n.T("validation.invalidKnowledgeCategoryID"), nil)
		}
		f.ParentCategoryID = &pid
	}
	if v := string(args.Peek("status")); v != "" {
		s := models.ArticleStatus(v)
		if s != models.ArticleStatusPublished && s != models.ArticleStatusDraft {
			return f, envelope.NewError(envelope.InputError, app.i18n.T("validation.invalidValue"), nil)
		}
		f.Status = &s
	}
	if v := string(args.Peek("sort")); v != "" {
		if v != knowledgebase.SortAsc && v != knowledgebase.SortDesc {
			return f, envelope.NewError(envelope.InputError, app.i18n.T("validation.invalidValue"), nil)
		}
		f.SortOrder = v
	}
	if v := string(args.Peek("date_from")); v != "" {
		t, perr := time.Parse(time.RFC3339, v)
		if perr != nil {
			return f, envelope.NewError(envelope.InputError, app.i18n.T("validation.invalidValue"), nil)
		}
		f.DateFrom = &t
	}
	if v := string(args.Peek("date_to")); v != "" {
		t, perr := time.Parse(time.RFC3339, v)
		if perr != nil {
			return f, envelope.NewError(envelope.InputError, app.i18n.T("validation.invalidValue"), nil)
		}
		f.DateTo = &t
	}
	if v := string(args.Peek("date_field")); v != "" {
		if v != knowledgebase.DateFieldCreated && v != knowledgebase.DateFieldUpdated {
			return f, envelope.NewError(envelope.InputError, app.i18n.T("validation.invalidValue"), nil)
		}
		f.DateField = v
	}
	if v := string(args.Peek("keyword_field")); v != "" {
		switch v {
		case knowledgebase.KeywordScopeAll, knowledgebase.KeywordScopeTitle, knowledgebase.KeywordScopeContent:
			f.KeywordScope = v
		default:
			return f, envelope.NewError(envelope.InputError, app.i18n.T("validation.invalidValue"), nil)
		}
	}
	if v := string(args.Peek("keywords")); v != "" {
		for _, kw := range strings.Fields(v) {
			f.Keywords = append(f.Keywords, kw)
		}
	}
	if v := string(args.Peek("page")); v != "" {
		f.Page, err = strconv.Atoi(v)
		if err != nil || f.Page < 1 {
			return f, envelope.NewError(envelope.InputError, app.i18n.T("validation.invalidValue"), nil)
		}
	}
	if v := string(args.Peek("per_page")); v != "" {
		f.PerPage, err = strconv.Atoi(v)
		if err != nil || f.PerPage < 1 {
			return f, envelope.NewError(envelope.InputError, app.i18n.T("validation.invalidValue"), nil)
		}
	}
	return f, nil
}

// buildKnowledgeArticleInput validates the request and turns it into the
// shape the manager expects.
func buildKnowledgeArticleInput(req *knowledgeArticleRequest) (knowledgebase.CreateArticleInput, error) {
	in := knowledgebase.CreateArticleInput{
		CategoryID:  req.CategoryID,
		ExternalID:  req.ExternalID,
		Code:        req.Code,
		Title:       strings.TrimSpace(req.Title),
		Author:      req.Author,
		Content:     req.Content,
		Status:      req.Status,
		Attachments: req.Attachments,
	}
	if in.Author != nil {
		v := strings.TrimSpace(*in.Author)
		if v == "" {
			in.Author = nil
		} else {
			in.Author = &v
		}
	}
	if in.Status == "" {
		in.Status = models.ArticleStatusPublished
	}
	if in.Title == "" {
		return in, envelope.NewError(envelope.InputError, "validation.knowledgeTitleRequired", nil)
	}
	if utf8.RuneCountInString(in.Title) > knowledgebase.MaxTitleLength {
		return in, envelope.NewError(envelope.InputError, "validation.knowledgeTitleTooLong", nil)
	}
	// Drafts are stored without a category so they show up under the
	// virtual "草稿" group in the sidebar; published articles must
	// always belong to a category.
	if in.Status == models.ArticleStatusPublished && (in.CategoryID == nil || *in.CategoryID == 0) {
		return in, envelope.NewError(envelope.InputError, "validation.knowledgeCategoryRequired", nil)
	}
	return in, nil
}

// validateKnowledgeCategoryName ensures a category name is present and
// within the allowed length.
func validateKnowledgeCategoryName(app *App, name string) error {
	if name == "" {
		return envelope.NewError(envelope.InputError, app.i18n.T("validation.knowledgeCategoryNameRequired"), nil)
	}
	if utf8.RuneCountInString(name) > knowledgebase.MaxCategoryNameLength {
		return envelope.NewError(envelope.InputError, app.i18n.T("validation.knowledgeCategoryNameTooLong"), nil)
	}
	return nil
}
