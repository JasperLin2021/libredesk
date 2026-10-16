package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/attachment"
	"github.com/abhinavxd/libredesk/internal/blacklist"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/fastglue"
)

const maxBlacklistReasonLength = 500

type addBlacklistReq struct {
	Reason string `json:"reason"`
}

// handleGetContactBlacklist returns the blacklist state of a single contact so
// the UI can render either "加入在线黑名单" or "移除在线黑名单".
func handleGetContactBlacklist(r *fastglue.Request) error {
	var (
		app          = r.Context.(*App)
		contactID, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
	)
	if contactID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	status, err := app.blacklist.Status(contactID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(status)
}

// handleAddContactBlacklist adds a contact to the online blacklist.
func handleAddContactBlacklist(r *fastglue.Request) error {
	var (
		app          = r.Context.(*App)
		contactID, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
		auser        = r.RequestCtx.UserValue("user").(amodels.User)
		req          = addBlacklistReq{}
	)
	if contactID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	if err := r.Decode(&req, "json"); err != nil {
		return sendErrorEnvelope(r, envelope.NewError(envelope.InputError, app.i18n.T("errors.parsingRequest"), nil))
	}
	req.Reason = strings.TrimSpace(req.Reason)
	if len([]rune(req.Reason)) > maxBlacklistReasonLength {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			app.i18n.Ts("globals.messages.maxLength", "max", strconv.Itoa(maxBlacklistReasonLength)), nil, envelope.InputError)
	}

	if err := app.blacklist.Add(contactID, auser.ID, req.Reason); err != nil {
		return sendErrorEnvelope(r, err)
	}

	status, err := app.blacklist.Status(contactID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(status)
}

// handleRemoveContactBlacklist removes a contact from the online blacklist.
func handleRemoveContactBlacklist(r *fastglue.Request) error {
	var (
		app          = r.Context.(*App)
		contactID, _ = strconv.Atoi(r.RequestCtx.UserValue("id").(string))
		auser        = r.RequestCtx.UserValue("user").(amodels.User)
	)
	if contactID <= 0 {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.InputError)
	}
	if err := app.blacklist.Remove(contactID, auser.ID); err != nil {
		return sendErrorEnvelope(r, err)
	}

	status, err := app.blacklist.Status(contactID)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}
	return r.SendEnvelope(status)
}

// handleGetBlacklistCustomers returns the paginated customer list for the
// "客户信息" page along with the number of currently blacklisted contacts.
// Supported query parameters: filter (all|blocked), search, period, start, end,
// order, order_by, page, page_size.
func handleGetBlacklistCustomers(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		arg = r.RequestCtx.QueryArgs()
	)
	page, pageSize := getPagination(r)
	from, to, err := blacklistTimeRange(arg)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	customers, total, blockedCount, err := app.blacklist.GetCustomers(blacklist.CustomerFilters{
		Filter:   strings.TrimSpace(string(arg.Peek("filter"))),
		Search:   string(arg.Peek("search")),
		From:     from,
		To:       to,
		Order:    strings.TrimSpace(string(arg.Peek("order"))),
		OrderBy:  strings.TrimSpace(string(arg.Peek("order_by"))),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(map[string]any{
		"results":       customers,
		"total":         total,
		"blocked_count": blockedCount,
		"per_page":      pageSize,
		"total_pages":   (total + pageSize - 1) / pageSize,
		"page":          page,
	})
}

// handleGetBlacklistLogs returns the paginated blacklist operation log for the
// "操作日志" page. Supported query parameters: action (add|remove), search,
// period, start, end, order, order_by, page, page_size.
func handleGetBlacklistLogs(r *fastglue.Request) error {
	var (
		app = r.Context.(*App)
		arg = r.RequestCtx.QueryArgs()
	)
	page, pageSize := getPagination(r)
	from, to, err := blacklistTimeRange(arg)
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	logs, total, err := app.blacklist.GetLogs(blacklist.LogFilters{
		Action:   strings.TrimSpace(string(arg.Peek("action"))),
		Search:   string(arg.Peek("search")),
		From:     from,
		To:       to,
		Order:    strings.TrimSpace(string(arg.Peek("order"))),
		OrderBy:  strings.TrimSpace(string(arg.Peek("order_by"))),
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	return r.SendEnvelope(map[string]any{
		"results":     logs,
		"total":       total,
		"per_page":    pageSize,
		"total_pages": (total + pageSize - 1) / pageSize,
		"page":        page,
	})
}

// blacklistTimeRange resolves the `period` / `start` / `end` query args shared
// by the customer and log listings into an optional time range.
func blacklistTimeRange(arg *fasthttp.Args) (*time.Time, *time.Time, error) {
	period := string(arg.Peek("period"))
	if period == "" {
		period = "all"
	}
	switch period {
	case "all":
		return nil, nil, nil
	case "today", "yesterday", "this_week", "this_month", "last_month":
		start, end, err := resolvePeriod(period)
		if err != nil {
			return nil, nil, err
		}
		return &start, &end, nil
	case "custom":
		start, end, err := parseCustomRange(arg.Peek("start"), arg.Peek("end"))
		if err != nil {
			return nil, nil, err
		}
		if start.IsZero() && end.IsZero() {
			return nil, nil, nil
		}
		if start.IsZero() {
			return nil, &end, nil
		}
		if end.IsZero() {
			return &start, nil, nil
		}
		return &start, &end, nil
	}
	return nil, nil, errBadFilter
}

// ── Widget interception ────────────────────────────────────────────────────
// A blacklisted contact never gets a real conversation: none is created, none
// is broadcast to the agent inbox and nothing lands in the service history.
// The widget only ever receives the configured welcome message.

// blockedConversationUUID returns a deterministic UUID for the placeholder
// conversation handed to a blacklisted visitor. It is stable so repeated
// widget inits for the same contact resolve to the same placeholder, which is
// what the widget's message cache keys on.
func blockedConversationUUID(contactID int) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(fmt.Sprintf("libredesk-blacklisted-contact:%d", contactID))).String()
}

// buildBlockedWidgetMessages renders the inbox welcome message — including the
// quick reply topic cards ("主题与问题") and the transfer-to-human button — as a
// widget-renderable chat message. It mirrors quickreply.Manager.SendWelcomeReply
// so a blacklisted visitor sees exactly what a normal visitor sees, minus the
// persisted conversation. It returns an empty slice when quick reply is
// disabled for the inbox or there is neither a welcome message nor any topic.
func buildBlockedWidgetMessages(app *App, inboxID int) []cmodels.ChatMessage {
	cfg, err := app.quickReply.GetConfig(inboxID)
	if err != nil {
		app.lo.Error("blacklist: error fetching quick reply config", "inbox_id", inboxID, "error", err)
		return []cmodels.ChatMessage{}
	}
	if !cfg.Enabled {
		return []cmodels.ChatMessage{}
	}

	// Topic cards: one button per configured topic, same as the real welcome
	// reply.
	items := make([]map[string]string, 0)
	if topics, err := app.quickReply.GetTopics(inboxID); err == nil {
		for _, topic := range topics {
			items = append(items, map[string]string{"label": topic.Name, "value": topic.Name})
		}
	} else {
		app.lo.Error("blacklist: error fetching quick reply topics", "inbox_id", inboxID, "error", err)
	}

	hasWelcomeMessage := strings.TrimSpace(cfg.WelcomeMessage) != ""
	hasTopics := len(items) > 0
	if !hasWelcomeMessage && !hasTopics {
		return []cmodels.ChatMessage{}
	}

	content := cfg.WelcomeMessage
	if !hasWelcomeMessage {
		content = ""
	}

	meta, err := json.Marshal(map[string]any{
		"type":             cmodels.MessageMetaTypeBotQuickReply,
		"items":            items,
		"transfer_keyword": cfg.TransferKeyword,
	})
	if err != nil {
		app.lo.Error("blacklist: error marshalling welcome meta", "inbox_id", inboxID, "error", err)
		meta = json.RawMessage(`{}`)
	}

	author := cmodels.MessageAuthor{FirstName: "System", Type: umodels.UserTypeAgent}
	if sys, err := app.user.GetSystemUser(); err == nil {
		author = cmodels.MessageAuthor{
			ID:                 sys.ID,
			FirstName:          sys.FirstName,
			LastName:           sys.LastName,
			AvatarURL:          sys.AvatarURL,
			AvailabilityStatus: sys.AvailabilityStatus,
			Type:               sys.Type,
		}
		app.conversation.SignAvatarURL(&author.AvatarURL)
	}

	now := time.Now()
	return []cmodels.ChatMessage{{
		UUID:        uuid.New().String(),
		Status:      cmodels.MessageStatusSent,
		CreatedAt:   now,
		Content:     content,
		TextContent: content,
		Author:      author,
		Meta:        meta,
	}}
}

// buildBlockedEchoMessage renders the message a blacklisted contact just sent
// back to their own widget so the bubble shows up on their side. Nothing is
// persisted and no automatic reply is generated: the message is purely an echo
// for the sender.
func buildBlockedEchoMessage(app *App, contactID int, content string) cmodels.ChatMessage {
	author := cmodels.MessageAuthor{
		ID:        contactID,
		FirstName: "我",
		Type:      umodels.UserTypeContact,
	}
	// Reflect the real identity (contact or visitor) so the widget renders the
	// bubble on the visitor's side of the thread.
	if contact, err := app.user.GetContactOrVisitor(contactID, ""); err == nil {
		author = cmodels.MessageAuthor{
			ID:                 contact.ID,
			FirstName:          contact.FirstName,
			LastName:           contact.LastName,
			Email:              contact.Email,
			AvatarURL:          contact.AvatarURL,
			AvailabilityStatus: contact.AvailabilityStatus,
			Type:               contact.Type,
			LastActiveAt:       contact.LastActiveAt,
		}
		app.conversation.SignAvatarURL(&author.AvatarURL)
	} else {
		app.lo.Error("blacklist: error loading contact for echo", "contact_id", contactID, "error", err)
	}

	now := time.Now()
	return cmodels.ChatMessage{
		UUID:           uuid.New().String(),
		Status:         cmodels.MessageStatusSent,
		CreatedAt:      now,
		Content:        content,
		TextContent:    content,
		Author:         author,
		Meta:           json.RawMessage(`{}`),
	}
}

// echoBlockedWidgetUpload stores a blacklisted contact's uploaded file and
// builds the echo message that makes the attachment show up in their own
// widget. The media is never linked to a conversation message, no reply is
// generated and no agent ever sees the file.
func echoBlockedWidgetUpload(app *App, r *fastglue.Request, form *multipart.Form, contactID int) (cmodels.ChatMessage, error) {
	config, err := getWidgetConfig(r)
	if err != nil {
		return cmodels.ChatMessage{}, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	if !config.Features.FileUpload {
		return cmodels.ChatMessage{}, envelope.NewError(envelope.InputError, app.i18n.T("status.disabledFileUpload"), nil)
	}

	files, ok := form.File["files"]
	if !ok || len(files) == 0 {
		return cmodels.ChatMessage{}, envelope.NewError(envelope.InputError, app.i18n.T("validation.notFoundFile"), nil)
	}

	fileHeader := files[0]
	file, err := fileHeader.Open()
	if err != nil {
		app.lo.Error("error reading uploaded file", "error", err)
		return cmodels.ChatMessage{}, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	defer file.Close()

	srcFileName := stringutil.SanitizeFilename(fileHeader.Filename)
	srcContentType := fileHeader.Header.Get("Content-Type")
	srcFileSize := fileHeader.Size
	srcExt := strings.TrimPrefix(strings.ToLower(filepath.Ext(srcFileName)), ".")

	if srcFileSize <= 0 {
		return cmodels.ChatMessage{}, envelope.NewError(envelope.InputError, app.i18n.T("media.fileEmpty"), nil)
	}

	consts := app.consts.Load().(*constants)
	if bytesToMegabytes(srcFileSize) > float64(consts.MaxFileUploadSizeMB) {
		return cmodels.ChatMessage{}, envelope.NewError(
			envelope.InputError,
			app.i18n.Ts("media.fileSizeTooLarge", "size", fmt.Sprintf("%dMB", consts.MaxFileUploadSizeMB)),
			nil,
		)
	}
	if !slices.Contains(consts.AllowedUploadFileExtensions, "*") && !slices.Contains(consts.AllowedUploadFileExtensions, srcExt) {
		return cmodels.ChatMessage{}, envelope.NewError(envelope.InputError, app.i18n.T("media.fileTypeNotAllowed"), nil)
	}

	fileContent, err := io.ReadAll(file)
	if err != nil {
		app.lo.Error("error reading file content", "error", err)
		return cmodels.ChatMessage{}, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	// Insert without a model link: nothing is persisted for a blacklisted
	// contact, and unlinked media is reaped by the media janitor.
	uploaded, err := app.media.UploadAndInsert(
		srcFileName,
		srcContentType,
		"",
		null.String{},
		null.Int{},
		bytes.NewReader(fileContent),
		int(srcFileSize),
		null.StringFrom(attachment.DispositionAttachment),
		nil,
	)
	if err != nil {
		app.lo.Error("error uploading media for blacklisted contact", "contact_id", contactID, "error", err)
		return cmodels.ChatMessage{}, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil)
	}

	echo := buildBlockedEchoMessage(app, contactID, "")
	echo.Attachments = attachment.Attachments{{
		Name:        srcFileName,
		Size:        int(srcFileSize),
		ContentType: srcContentType,
		Disposition: attachment.DispositionAttachment,
		UUID:        uploaded.UUID,
		URL:         app.media.GetURL(uploaded.UUID, srcContentType, srcFileName),
	}}
	return echo, nil
}

// isBlacklistedContact reports whether the widget session's contact is on the
// online blacklist. Errors are logged and treated as "not blacklisted" so a
// transient DB problem never locks a legitimate visitor out.
func isBlacklistedContact(app *App, contactID int) bool {
	if contactID <= 0 {
		return false
	}
	blocked, err := app.blacklist.IsBlocked(contactID)
	if err != nil {
		app.lo.Error("error checking contact blacklist", "contact_id", contactID, "error", err)
		return false
	}
	return blocked
}
