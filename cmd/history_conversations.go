package main

import (
	"strings"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/conversation"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/zerodha/fastglue"
)

// handleGetHistoryConversations returns the paginated service history list of
// conversation close events ("会话记录"): one record per time a conversation was
// set to "Closed".
// Supported query parameters:
//
//	page, page_size            – pagination
//	order                      – asc | desc, applied to the selected time dimension
//	time_dimension             – first_reply (pick up time, default) | closed (close event time)
//	period                     – today | yesterday | this_week | this_month | last_month | custom | all
//	start, end                 – explicit range, used when period=custom
//	visitor_id                 – contact external id, exact match
func handleGetHistoryConversations(r *fastglue.Request) error {
	var (
		app           = r.Context.(*App)
		user          = r.RequestCtx.UserValue("user").(amodels.User)
		args          = r.RequestCtx.QueryArgs()
		order         = strings.ToLower(strings.TrimSpace(string(args.Peek("order"))))
		timeDimension = strings.ToLower(strings.TrimSpace(string(args.Peek("time_dimension"))))
	)

	if order != "" && order != "asc" && order != "desc" {
		return sendErrorEnvelope(r, errBadFilter)
	}
	if timeDimension != "" && timeDimension != "first_reply" && timeDimension != "closed" {
		return sendErrorEnvelope(r, errBadFilter)
	}

	page, pageSize := getPagination(r)
	filters := conversation.HistoryConversationsFilter{
		VisitorID:     strings.TrimSpace(string(args.Peek("visitor_id"))),
		Order:         order,
		TimeDimension: timeDimension,
		Page:          page,
		PageSize:      pageSize,
	}

	switch period := string(args.Peek("period")); period {
	case "", "all":
		// No time bound.
	case "today", "yesterday", "this_week", "this_month", "last_month":
		start, end, err := resolvePeriod(period)
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		filters.Start = &start
		filters.End = &end
	case "custom":
		start, end, err := parseCustomRange(args.Peek("start"), args.Peek("end"))
		if err != nil {
			return sendErrorEnvelope(r, err)
		}
		if !start.IsZero() {
			filters.Start = &start
		}
		if !end.IsZero() {
			filters.End = &end
		}
	default:
		return sendErrorEnvelope(r, errBadFilter)
	}

	conversations, err := app.conversation.GetHistoryConversations(user.ID, filters)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	total := 0
	if len(conversations) > 0 {
		total = conversations[0].Total
	}

	return r.SendEnvelope(envelope.PageResults{
		Results:    conversations,
		Total:      total,
		PerPage:    pageSize,
		TotalPages: (total + pageSize - 1) / pageSize,
		Page:       page,
	})
}
