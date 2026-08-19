package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/abhinavxd/libredesk/internal/envelope"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	"github.com/valyala/fasthttp"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/fastglue"
)

const bixiaocrmAPIURL = "https://api-app.bixiaocrm.com/personnel/index.json"

// bixiaoAuthRequest is the request body from the widget.
type bixiaoAuthRequest struct {
	Token        string `json:"token"`
	Device       string `json:"device"`
	VersionSeq   string `json:"version_seq"`
	Inbox        string `json:"inbox"`
	VisitorToken string `json:"visitor_token"`
}

// bixiaoUserResponse is the response from bixiaocrm's personnel API.
// It supports both flat and wrapped response formats.
type bixiaoUserResponse struct {
	Nickname string `json:"nickname"`
	TlID     string `json:"tlId"`
	Phone    string `json:"phone"`
	Avatar   string `json:"avatar"`
}

// bixiaoAPIResponse handles possible wrapper formats from the bixiaocrm API.
type bixiaoAPIResponse struct {
	Code int                `json:"code"`
	Data bixiaoUserResponse `json:"data"`
}

// handleBixiaoAuth handles third-party authentication via bixiaocrm.
// It calls the bixiaocrm API to verify user credentials, then creates or
// updates a contact and issues a session token for the chat widget.
func handleBixiaoAuth(r *fastglue.Request) error {
	app := r.Context.(*App)

	// CORS header for the actual POST response.
	r.RequestCtx.Response.Header.Set("Access-Control-Allow-Origin", "*")

	inbox, err := getWidgetInbox(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError,
			app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	config, err := getWidgetConfig(r)
	if err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError,
			app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	var req bixiaoAuthRequest
	if err := r.Decode(&req, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}

	// Validate required fields.
	if req.Token == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			app.i18n.Ts("globals.messages.required", "name", "token"), nil, envelope.InputError)
	}
	if req.Device == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			app.i18n.Ts("globals.messages.required", "name", "device"), nil, envelope.InputError)
	}
	if req.VersionSeq == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest,
			app.i18n.Ts("globals.messages.required", "name", "version_seq"), nil, envelope.InputError)
	}

	// Call the bixiaocrm personnel API.
	bixiaoUser, rawBody, err := fetchBixiaoUser(req.Token, req.Device, req.VersionSeq)
	if err != nil {
		app.lo.Error("error calling bixiaocrm API", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized,
			app.i18n.T("globals.terms.unAuthorized"), nil, envelope.UnauthorizedError)
	}

	if bixiaoUser.TlID == "" {
		app.lo.Error("bixiaocrm API returned empty tlId", "body", rawBody, "nickname", bixiaoUser.Nickname)
		return r.SendErrorEnvelope(fasthttp.StatusUnauthorized,
			app.i18n.T("globals.terms.unAuthorized"), nil, envelope.UnauthorizedError)
	}

	// Resolve or create the contact using tlId as external_user_id.
	contactID, err := resolveOrCreateBixiaoContact(app, bixiaoUser)
	if err != nil {
		app.lo.Error("error resolving bixiao contact", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError,
			app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	// Store tlId in custom_attributes for reference.
	customAttrs := map[string]any{"tl_id": bixiaoUser.TlID}
	if err := app.user.SaveCustomAttributes(contactID, customAttrs, false); err != nil {
		app.lo.Error("error saving bixiao custom attributes", "contact_id", contactID, "error", err)
	}

	// Store external_user_id as well for reverse lookup.
	if err := app.user.SaveCustomAttributes(contactID, map[string]any{"external_user_id": bixiaoUser.TlID}, false); err != nil {
		app.lo.Error("error saving external_user_id in custom attributes", "contact_id", contactID, "error", err)
	}

	// Merge visitor to contact if visitor token is provided.
	// The widget stores the database visitor_token (a UUID) in its cookie,
	// not the Redis session token, so fall back to a DB lookup when the Redis
	// session is not found.
	app.lo.Debug("bixiao auth request", "visitor_token_provided", req.VisitorToken != "", "visitor_token_len", len(req.VisitorToken))
	if req.VisitorToken != "" {
		var visitorID int
		if visitorSession, vErr := loadSession(app, req.VisitorToken, config); vErr == nil &&
			visitorSession.IsVisitor && visitorSession.UserID > 0 &&
			visitorSession.UserID != contactID && visitorSession.InboxID == inbox.ID {
			visitorID = visitorSession.UserID
			app.lo.Debug("bixiao auth found visitor via redis session", "visitor_id", visitorID)
		} else if dbVisitor, dbErr := app.user.GetVisitorByToken(req.VisitorToken); dbErr == nil &&
			dbVisitor.ID > 0 && dbVisitor.ID != contactID {
			visitorID = dbVisitor.ID
			app.lo.Debug("bixiao auth found visitor via db lookup", "visitor_id", visitorID)
		} else {
			// Neither the Redis session nor the DB lookup resolved a visitor
			// for this token, so there is nothing to delete.
			app.lo.Debug("bixiao auth visitor lookup found nothing",
				"visitor_token", req.VisitorToken)
		}

		if visitorID > 0 {
			// Delete the visitor and their conversations instead of merging
			// them into the authenticated contact, so the user starts with a
			// clean conversation list after authenticating.
			if err := app.user.DeleteVisitor(visitorID); err != nil {
				app.lo.Error("error deleting visitor", "visitor_id", visitorID, "contact_id", contactID, "error", err)
			} else {
				app.lo.Info("deleted visitor after bixiao auth", "visitor_id", visitorID, "contact_id", contactID)
				deleteSessionToken(app, req.VisitorToken)
				// Signal frontend to clear visitor token cookie.
				r.RequestCtx.Response.Header.Set(hdrClearVisitorToken, "true")
			}
		}
	}

	// Generate session token.
	ctx := context.Background()
	reverseKey := fmt.Sprintf("widget_user:%d:%d", inbox.ID, contactID)
	sessionTTL := getSessionDuration(config)

	sendSession := func(token string) error {
		app.redis.Set(ctx, reverseKey, token, sessionTTL)
		userInfo := map[string]any{
			"user_id":    contactID,
			"is_visitor": false,
			"first_name": bixiaoUser.Nickname,
			"last_name":  "",
		}
		if bixiaoUser.Avatar != "" {
			userInfo["avatar"] = bixiaoUser.Avatar
		}
		return r.SendEnvelope(map[string]any{
			"session_token": token,
			"user":          userInfo,
		})
	}

	// Check for an existing session.
	existingToken, redisErr := app.redis.Get(ctx, reverseKey).Result()
	if redisErr == nil && existingToken != "" {
		if isValidSession(ctx, app, existingToken, inbox, contactID) {
			return sendSession(existingToken)
		}
	}

	token, err := generateSessionToken(app, contactID, inbox.ID, false, bixiaoUser.TlID, sessionTTL)
	if err != nil {
		app.lo.Error("error generating session token for bixiao auth", "error", err)
		return r.SendErrorEnvelope(fasthttp.StatusInternalServerError,
			app.i18n.T("globals.messages.somethingWentWrong"), nil, envelope.GeneralError)
	}

	return sendSession(token)
}

// fetchBixiaoUser makes an HTTP POST to the bixiaocrm personnel API
// and parses the user info from the response.
// Returns the parsed user, raw body for debug logging, and any error.
func fetchBixiaoUser(token, device, versionSeq string) (bixiaoUserResponse, string, error) {
	var result bixiaoUserResponse

	httpReq, err := http.NewRequest(http.MethodPost, bixiaocrmAPIURL, nil)
	if err != nil {
		return result, "", fmt.Errorf("creating bixiaocrm request: %w", err)
	}

	httpReq.Header.Set("token", token)
	httpReq.Header.Set("device", device)
	httpReq.Header.Set("versionSeq", versionSeq)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return result, "", fmt.Errorf("calling bixiaocrm API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, "", fmt.Errorf("reading bixiaocrm response: %w", err)
	}
	rawBody := string(body)

	if resp.StatusCode != http.StatusOK {
		return result, rawBody, fmt.Errorf("bixiaocrm API returned status %d: %s", resp.StatusCode, rawBody)
	}

	// Try wrapped format first: { "code": ..., "data": { "nickname": "...", "tlId": "..." } }
	var wrappedResp bixiaoAPIResponse
	if err := json.Unmarshal(body, &wrappedResp); err == nil && wrappedResp.Data.TlID != "" {
		return wrappedResp.Data, rawBody, nil
	}

	// Try flat format: { "nickname": "...", "tlId": "..." }
	if err := json.Unmarshal(body, &result); err != nil {
		return result, rawBody, fmt.Errorf("parsing bixiaocrm response: %w", err)
	}

	return result, rawBody, nil
}

// resolveOrCreateBixiaoContact finds or creates a contact based on the bixiaocrm user info.
func resolveOrCreateBixiaoContact(app *App, user bixiaoUserResponse) (int, error) {
	// Try to find existing contact by external_user_id (tlId).
	existing, err := app.user.GetByExternalID(user.TlID)
	if err == nil {
		// Contact exists - sync nickname, phone and avatar if they changed.
		if user.Nickname != "" && existing.FirstName != user.Nickname {
			if updateErr := app.user.UpdateContactBasicInfo(existing.ID, user.Nickname, "", "", "", ""); updateErr != nil {
				app.lo.Error("error updating bixiao contact name", "contact_id", existing.ID, "error", updateErr)
			}
		}
		if user.Phone != "" && existing.PhoneNumber.String != user.Phone {
			if updateErr := app.user.UpdateContactBasicInfo(existing.ID, "", "", "", user.Phone, ""); updateErr != nil {
				app.lo.Error("error updating bixiao contact phone", "contact_id", existing.ID, "error", updateErr)
			}
		}
		if user.Avatar != "" && existing.AvatarURL.String != user.Avatar {
			if updateErr := app.user.UpdateAvatar(existing.ID, user.Avatar); updateErr != nil {
				app.lo.Error("error updating bixiao contact avatar", "contact_id", existing.ID, "error", updateErr)
			}
		}
		return existing.ID, nil
	}

	if envErr, ok := err.(envelope.Error); !ok || envErr.ErrorType != envelope.NotFoundError {
		return 0, err
	}

	// Create new contact.
	newUser := umodels.User{
		FirstName:      user.Nickname,
		ExternalUserID: null.NewString(user.TlID, true),
		PhoneNumber:    null.NewString(user.Phone, user.Phone != ""),
		AvatarURL:      null.NewString(user.Avatar, user.Avatar != ""),
	}

	if createErr := app.user.CreateContact(&newUser); createErr != nil {
		app.lo.Error("error creating bixiao contact", "error", createErr)
		return 0, createErr
	}

	return newUser.ID, nil
}

// isValidSession checks if a given session token is still valid for the inbox and contact.
func isValidSession(ctx context.Context, app *App, token string, inbox imodels.Inbox, contactID int) bool {
	key := widgetSessionPrefix + token
	userIDStr, err := app.redis.HGet(ctx, key, "user_id").Result()
	if err != nil || userIDStr == "" {
		return false
	}
	inboxIDStr, err := app.redis.HGet(ctx, key, "inbox_id").Result()
	if err != nil {
		return false
	}
	var uid, iid int
	if _, scanErr := fmt.Sscanf(userIDStr, "%d", &uid); scanErr != nil {
		return false
	}
	if _, scanErr := fmt.Sscanf(inboxIDStr, "%d", &iid); scanErr != nil {
		return false
	}
	return uid == contactID && iid == inbox.ID
}
