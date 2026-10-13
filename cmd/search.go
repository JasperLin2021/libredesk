package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	smodels "github.com/abhinavxd/libredesk/internal/search/models"
	"github.com/zerodha/fastglue"
)

const (
	minSearchQueryLength = 3
)

var errBadFilter = envelope.NewError(envelope.InputError, "invalid search filter", nil)

// handleSearchConversations searches conversations based on the query.
func handleSearchConversations(r *fastglue.Request) error {
	app, user, q, filters, err := searchInputs(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	results, err := app.search.Conversations(q, filters)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	uuids := make([]string, len(results))
	for i, c := range results {
		uuids[i] = c.UUID
	}
	allowed, err := app.conversation.FilterAuthorizedListUUIDs(user.ID, uuids)
	if err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}
	set := uuidSet(allowed)
	out := make([]smodels.ConversationResult, 0, len(allowed))
	for _, c := range results {
		if _, ok := set[c.UUID]; ok {
			out = append(out, c)
		}
	}
	return r.SendEnvelope(out)
}

// handleSearchMessages searches messages based on the query.
func handleSearchMessages(r *fastglue.Request) error {
	app, user, q, filters, err := searchInputs(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	results, err := app.search.Messages(q, filters)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	uuids := make([]string, len(results))
	for i, m := range results {
		uuids[i] = m.ConversationUUID
	}
	allowed, err := app.conversation.FilterAuthorizedListUUIDs(user.ID, uuids)
	if err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}
	set := uuidSet(allowed)
	out := make([]smodels.MessageResult, 0, len(allowed))
	for _, m := range results {
		if _, ok := set[m.ConversationUUID]; ok {
			out = append(out, m)
		}
	}
	return r.SendEnvelope(out)
}

// handleSearchContacts searches contacts based on the query.
func handleSearchContacts(r *fastglue.Request) error {
	app, _, q, filters, err := searchInputs(r)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	// Contacts do not honour the optional filters, but the signature is shared
	// to keep the call sites uniform.
	_ = filters
	results, err := app.search.Contacts(q)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(results)
}

func searchInputs(r *fastglue.Request) (*App, amodels.User, string, smodels.Filters, error) {
	app := r.Context.(*App)
	user, _ := r.RequestCtx.UserValue("user").(amodels.User)
	q := string(r.RequestCtx.QueryArgs().Peek("query"))
	if len(q) < minSearchQueryLength {
		return app, user, "", smodels.Filters{}, envelope.NewError(envelope.InputError, app.i18n.Ts("search.minQueryLength", "length", fmt.Sprintf("%d", minSearchQueryLength)), nil)
	}
	filters, err := parseSearchFilters(r, app)
	if err != nil {
		return app, user, "", smodels.Filters{}, err
	}
	return app, user, q, filters, nil
}

// parseSearchFilters extracts the optional filter parameters from the request.
// All parameters are optional; missing parameters leave the corresponding
// filter unset so the SQL `IS NULL` skip-pattern matches every row.
func parseSearchFilters(r *fastglue.Request, app *App) (smodels.Filters, error) {
	args := r.RequestCtx.QueryArgs()
	filters := smodels.Filters{
		Sort: string(args.Peek("sort")),
	}
	if filters.Sort != "" && filters.Sort != "newest" && filters.Sort != "oldest" {
		return smodels.Filters{}, errBadFilter
	}

	// Time range: either an explicit `period` (today/yesterday/this_week/...)
	// or a custom `start`/`end` pair. `custom` falls back to start/end.
	period := string(args.Peek("period"))
	switch period {
	case "", "all":
		// no time bound
	case "today", "yesterday", "this_week", "this_month", "last_month":
		start, end, err := resolvePeriod(period)
		if err != nil {
			return smodels.Filters{}, err
		}
		filters.StartAt = start
		filters.EndAt = end
	case "custom":
		start, end, err := parseCustomRange(args.Peek("start"), args.Peek("end"))
		if err != nil {
			return smodels.Filters{}, err
		}
		filters.StartAt = start
		filters.EndAt = end
	default:
		return smodels.Filters{}, errBadFilter
	}

	// Inbox filter – comma-separated list of inbox ids.
	if raw := string(args.Peek("inbox_ids")); raw != "" {
		for _, part := range strings.Split(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			id, err := strconv.Atoi(part)
			if err != nil || id <= 0 {
				return smodels.Filters{}, errBadFilter
			}
			filters.InboxIDs = append(filters.InboxIDs, id)
		}
	}

	// Agent filter – single agent (assigned user) id.
	if raw := string(args.Peek("agent_id")); raw != "" {
		id, err := strconv.Atoi(raw)
		if err != nil || id < 0 {
			return smodels.Filters{}, errBadFilter
		}
		filters.AgentID = id
	}

	return filters, nil
}

// resolvePeriod maps a named period to its [start, end] timestamp pair in
// the local timezone. Both bounds are inclusive.
func resolvePeriod(period string) (time.Time, time.Time, error) {
	now := time.Now()
	loc := now.Location()
	switch period {
	case "today":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		end := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), loc)
		return start, end, nil
	case "yesterday":
		y := now.AddDate(0, 0, -1)
		start := time.Date(y.Year(), y.Month(), y.Day(), 0, 0, 0, 0, loc)
		end := time.Date(y.Year(), y.Month(), y.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), loc)
		return start, end, nil
	case "this_week":
		// Week starts on Monday.
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		monday := now.AddDate(0, 0, -(weekday - 1))
		start := time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, loc)
		end := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), loc)
		return start, end, nil
	case "this_month":
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		end := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), loc)
		return start, end, nil
	case "last_month":
		firstOfThisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		lastOfPrevMonth := firstOfThisMonth.AddDate(0, 0, -1)
		start := time.Date(lastOfPrevMonth.Year(), lastOfPrevMonth.Month(), 1, 0, 0, 0, 0, loc)
		end := time.Date(lastOfPrevMonth.Year(), lastOfPrevMonth.Month(), lastOfPrevMonth.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), loc)
		return start, end, nil
	}
	return time.Time{}, time.Time{}, errBadFilter
}

// parseCustomRange parses the optional `start` and `end` query parameters.
// The accepted layouts mirror what the front end sends from the period
// dropdown ("YYYY/MM/DD HH:mm:ss") plus the common RFC3339 form.
var customRangeLayouts = []string{
	"2006/01/02 15:04:05",
	"2006/01/02 15:04",
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
	time.RFC3339,
	// datetime-local inputs submit "YYYY-MM-DDTHH:mm" (no seconds, no zone).
	"2006-01-02T15:04:05",
	"2006-01-02T15:04",
}

func parseCustomRange(startRaw, endRaw []byte) (time.Time, time.Time, error) {
	var start, end time.Time
	if len(startRaw) > 0 {
		t, err := parseTimeString(string(startRaw))
		if err != nil {
			return time.Time{}, time.Time{}, errBadFilter
		}
		start = t
	}
	if len(endRaw) > 0 {
		t, err := parseTimeString(string(endRaw))
		if err != nil {
			return time.Time{}, time.Time{}, errBadFilter
		}
		end = t
	}
	if !start.IsZero() && !end.IsZero() && end.Before(start) {
		return time.Time{}, time.Time{}, errBadFilter
	}
	return start, end, nil
}

func parseTimeString(s string) (time.Time, error) {
	for _, layout := range customRangeLayouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported time format: %q", s)
}

func uuidSet(uuids []string) map[string]struct{} {
	s := make(map[string]struct{}, len(uuids))
	for _, u := range uuids {
		s[u] = struct{}{}
	}
	return s
}
