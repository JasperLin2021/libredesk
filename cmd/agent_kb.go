package main

import (
	"strconv"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/knowledgebase/models"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

// handleListAgentKbArticles returns a paginated list of published knowledge
// articles for the agent-facing "知识库" sidebar tab. It reuses the admin
// filter parser (keyword_field / keywords / page / per_page) but forces
// status=published so drafts never leak to regular agents.
func handleListAgentKbArticles(r *fastglue.Request) error {
	app := r.Context.(*App)
	filter, err := parseKnowledgeArticleFilter(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	published := models.ArticleStatusPublished
	filter.Status = &published
	res, err := app.knowledgeBase.List(filter)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(res)
}

// handleGetAgentKbCategories returns the two-level knowledge base category
// tree for the agent-facing "知识库" sidebar filter. It is read-only and
// gated by `messages:read` (instead of the admin-only `knowledge_base:manage`)
// so any agent who can handle conversations can filter by category.
func handleGetAgentKbCategories(r *fastglue.Request) error {
	app := r.Context.(*App)
	tree, err := app.knowledgeBase.GetTree()
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(tree)
}

// handleGetAgentKbArticle returns a single published knowledge article with
// its full content. Used by the "发送正文" action, which needs the complete
// body (the list endpoint only returns a summary). Drafts are reported as
// not found so they stay invisible to agents.
func handleGetAgentKbArticle(r *fastglue.Request) error {
	app := r.Context.(*App)
	id, err := strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	if err != nil || id == 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("validation.invalidKnowledgeArticleID"), nil, envelope.InputError)
	}
	article, err := app.knowledgeBase.GetArticle(id)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	if article.Status != models.ArticleStatusPublished {
		return r.SendErrorEnvelope(fasthttp.StatusNotFound, app.i18n.T("validation.notFoundKnowledgeArticle"), nil, envelope.InputError)
	}
	return r.SendEnvelope(article)
}
