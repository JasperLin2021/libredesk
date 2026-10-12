package main

import (
	amodels "github.com/abhinavxd/libredesk/internal/auth/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
	realip "github.com/ferluci/fast-realip"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// handleLogin logs in the user and returns the user.
func handleLogin(r *fastglue.Request) error {
	var (
		app      = r.Context.(*App)
		ip       = realip.FromRequest(r.RequestCtx)
		loginReq loginRequest
	)

	// Decode JSON request.
	if err := r.Decode(&loginReq, "json"); err != nil {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("errors.parsingRequest"), nil, envelope.InputError)
	}

	if loginReq.Email == "" || loginReq.Password == "" {
		return r.SendErrorEnvelope(fasthttp.StatusBadRequest, app.i18n.T("globals.messages.badRequest"), nil, envelope.InputError)
	}

	// Verify email and password.
	user, err := app.user.VerifyPassword(loginReq.Email, []byte(loginReq.Password))
	if err != nil {
		return sendErrorEnvelope(r, err)
	}

	// Check if user is enabled.
	if !user.Enabled {
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("user.accountDisabled"), nil))
	}

	if err := app.auth.SaveSession(amodels.User{
		ID:        user.ID,
		Email:     user.Email.String,
		FirstName: user.FirstName,
		LastName:  user.LastName,
	}, r); err != nil {
		app.lo.Error("error saving session", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}
	// Set CSRF cookie if not already set.
	if err := app.auth.SetCSRFCookie(r); err != nil {
		app.lo.Error("error setting csrf cookie", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}

	// Update last login time.
	if err := app.user.UpdateLastLoginAt(user.ID); err != nil {
		return sendErrorEnvelope(r, err)
	}

	app.user.InvalidateAgentCache(user.ID)

	// Insert activity log.
	if err := app.activityLog.Login(user.ID, user.Email.String, ip); err != nil {
		app.lo.Error("error creating login activity log", "error", err)
	}

	return r.SendEnvelope(user)
}

// handleLogout logs out the user and redirects to the dashboard.
//
// NOTE: Prefer handleLogoutAPI for SPA clients. This handler exists for
// backwards compatibility with bookmarks / curl / direct GET navigation.
// The 302 redirect uses a relative path "/" which, when the request was
// proxied to the backend (e.g. via Vite dev server) is resolved against
// the backend URL rather than the SPA, so it is not reliable as a UX.
func handleLogout(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		ip    = realip.FromRequest(r.RequestCtx)
	)

	// Insert activity log.
	if err := app.activityLog.Logout(auser.ID, auser.Email, ip); err != nil {
		app.lo.Error("error creating logout activity log", "error", err)
	}

	if err := app.auth.DestroySession(r); err != nil {
		app.lo.Error("error destroying session", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}
	// Add no-cache headers.
	r.RequestCtx.Response.Header.Add("Cache-Control",
		"no-store, no-cache, must-revalidate, post-check=0, pre-check=0")
	r.RequestCtx.Response.Header.Add("Pragma", "no-cache")
	r.RequestCtx.Response.Header.Add("Expires", "-1")
	return r.RedirectURI("/", fasthttp.StatusFound, nil, "")
}

// handleLogoutAPI handles API-based logout (POST /api/v1/auth/logout).
//
// It only destroys the server-side session and returns 200 OK. The SPA
// is responsible for clearing the local user store and navigating to the
// login screen via the in-app router, which keeps the browser on the
// SPA origin instead of a 302 redirect to the backend URL.
func handleLogoutAPI(r *fastglue.Request) error {
	var (
		app   = r.Context.(*App)
		auser = r.RequestCtx.UserValue("user").(amodels.User)
		ip    = realip.FromRequest(r.RequestCtx)
	)

	// Insert activity log.
	if err := app.activityLog.Logout(auser.ID, auser.Email, ip); err != nil {
		app.lo.Error("error creating logout activity log", "error", err)
	}

	if err := app.auth.DestroySession(r); err != nil {
		app.lo.Error("error destroying session", "error", err)
		return sendErrorEnvelope(r, envelope.NewError(envelope.GeneralError, app.i18n.T("globals.messages.somethingWentWrong"), nil))
	}
	return r.SendEnvelope(true)
}
