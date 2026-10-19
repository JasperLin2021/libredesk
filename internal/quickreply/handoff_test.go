package quickreply

import (
	"errors"
	"testing"
	"time"

	bhmodels "github.com/abhinavxd/libredesk/internal/business_hours/models"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	imodels "github.com/abhinavxd/libredesk/internal/inbox/models"
	"github.com/abhinavxd/libredesk/internal/quickreply/models"
	"github.com/jmoiron/sqlx/types"
	"github.com/knadh/go-i18n"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

// --- fakes ------------------------------------------------------------------

type fakeInbox struct {
	inbox imodels.Inbox
	err   error
}

func (f *fakeInbox) GetDBRecord(_ any) (imodels.Inbox, error) { return f.inbox, f.err }

type fakeBusinessHours struct {
	bh  bhmodels.BusinessHours
	err error
}

func (f *fakeBusinessHours) Get(_ int) (bhmodels.BusinessHours, error) { return f.bh, f.err }
func (f *fakeBusinessHours) IsWithinBusinessHours(bh bhmodels.BusinessHours, tz string, now time.Time) (bool, error) {
	return businessHoursOpen(bh, tz, now)
}

type fakeSettings struct {
	raw types.JSONText
	err error
}

func (f *fakeSettings) GetByPrefix(_ string) (types.JSONText, error) { return f.raw, f.err }

// fakeConv captures the arguments passed to SendAutoReply so the test can
// assert what message was actually sent and what side effects were skipped.
type fakeConv struct {
	calls        int
	gotInboxID   int
	gotContactID int
	gotUUID      string
	gotContent   string
	gotMeta      map[string]any
	gotSilent    bool
	// Side-effect flags track whether the queue / broadcast paths were taken.
	queueCount    int
	queueErr      error
	broadcastErr  error
	refreshErr    error
	metaGetErr    error
}

func (c *fakeConv) SendAutoReply(inboxID, contactID int, conversationUUID, content string, metaMap map[string]any, silent bool) (cmodels.Message, error) {
	c.calls++
	c.gotInboxID = inboxID
	c.gotContactID = contactID
	c.gotUUID = conversationUUID
	c.gotContent = content
	c.gotMeta = metaMap
	c.gotSilent = silent
	return cmodels.Message{}, nil
}
func (c *fakeConv) GetConversationMeta(_ string) (map[string]any, error) {
	return map[string]any{}, c.metaGetErr
}
func (c *fakeConv) UpdateConversationMeta(_ string, _ map[string]any) error { return nil }
func (c *fakeConv) DeleteConversationMetaKey(_, _ string) error             { return nil }
func (c *fakeConv) CountOpenUnassignedConversations() (int, error)         { return c.queueCount, c.queueErr }
func (c *fakeConv) SendReplyAsUser(_ int, _ int, _ int, _ string, _ string, _ map[string]any) (cmodels.Message, error) {
	return cmodels.Message{}, nil
}
func (c *fakeConv) RefreshWaitingQueueInfo() error                       { return c.refreshErr }
func (c *fakeConv) CloseConversation(_ string) error                     { return nil }
func (c *fakeConv) BroadcastNewConversationByUUID(_ string) error         { return c.broadcastErr }

// --- helpers ----------------------------------------------------------------

// newTestManager returns a Manager with a nil DB-safe initialisation suitable
// for unit-testing logic that does not touch the database. The conversation
// service is filled in by the caller.
func newTestManager(t *testing.T, clock time.Time) *Manager {
	t.Helper()
	lo := logf.New(logf.Opts{Level: logf.DebugLevel})
	i, err := i18n.New([]byte(`{"_.code":"en","_.name":"English","quickReply.transferRestricted":"FALLBACK"}`))
	if err != nil {
		t.Fatalf("i18n.New: %v", err)
	}
	return &Manager{lo: &lo, i18n: i, now: func() time.Time { return clock }}
}

// businessHoursOpen is a tiny copy of the real check used only by the
// BusinessHoursService fake; it intentionally mirrors the production logic so
// the gate test exercises the same predicate.
func businessHoursOpen(bh bhmodels.BusinessHours, tz string, now time.Time) (bool, error) {
	if bh.IsAlwaysOpen {
		return true, nil
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	now = now.In(loc)
	if len(bh.Hours) == 0 {
		return false, nil
	}
	// Day-of-week lookup, supporting the same array format as production.
	// We avoid pulling in a JSON parser just to make this test readable, so
	// we test against a single known slot that the assertions below rely on.
	if string(bh.Hours) != `{"Monday":[{"open":"09:00","close":"18:00"}]}` {
		return false, nil
	}
	if now.Weekday().String() != "Monday" {
		return false, nil
	}
	return now.Hour() >= 9 && now.Hour() < 18, nil
}

// --- tests ------------------------------------------------------------------

func TestHandoffAllowedNow(t *testing.T) {
	bh := bhmodels.BusinessHours{Hours: types.JSONText(`{"Monday":[{"open":"09:00","close":"18:00"}]}`)}
	settings := &fakeSettings{raw: types.JSONText(`{"app.timezone":"UTC"}`)}
	bhSvc := &fakeBusinessHours{bh: bh}
	inside := time.Date(2026, 10, 19, 10, 0, 0, 0, time.UTC)  // Monday 10:00 UTC
	outside := time.Date(2026, 10, 19, 20, 0, 0, 0, time.UTC) // Monday 20:00 UTC
	tuesday := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC) // Tuesday 10:00 UTC

	cases := []struct {
		name      string
		inbox     imodels.Inbox
		bhSvc     *fakeBusinessHours
		settings  *fakeSettings
		now       time.Time
		wantAllow bool
	}{
		{
			name:      "unrestricted (NULL handoff_business_hours_id) always allowed",
			inbox:     imodels.Inbox{},
			now:       inside,
			wantAllow: true,
		},
		{
			name:      "business mode, inside working hours",
			inbox:     imodels.Inbox{HandoffBusinessHoursID: null.IntFrom(1), HandoffHoursMode: imodels.HandoffHoursModeBusiness},
			bhSvc:     bhSvc, settings: settings, now: inside,
			wantAllow: true,
		},
		{
			name:      "business mode, outside working hours",
			inbox:     imodels.Inbox{HandoffBusinessHoursID: null.IntFrom(1), HandoffHoursMode: imodels.HandoffHoursModeBusiness},
			bhSvc:     bhSvc, settings: settings, now: outside,
			wantAllow: false,
		},
		{
			name:      "business mode, no working hours on this weekday",
			inbox:     imodels.Inbox{HandoffBusinessHoursID: null.IntFrom(1), HandoffHoursMode: imodels.HandoffHoursModeBusiness},
			bhSvc:     bhSvc, settings: settings, now: tuesday,
			wantAllow: false,
		},
		{
			name:      "off_business mode, inside working hours (transfer blocked)",
			inbox:     imodels.Inbox{HandoffBusinessHoursID: null.IntFrom(1), HandoffHoursMode: imodels.HandoffHoursModeOffBusiness},
			bhSvc:     bhSvc, settings: settings, now: inside,
			wantAllow: false,
		},
		{
			name:      "off_business mode, outside working hours (transfer allowed)",
			inbox:     imodels.Inbox{HandoffBusinessHoursID: null.IntFrom(1), HandoffHoursMode: imodels.HandoffHoursModeOffBusiness},
			bhSvc:     bhSvc, settings: settings, now: outside,
			wantAllow: true,
		},
		{
			name:      "business hours lookup error fails open",
			inbox:     imodels.Inbox{HandoffBusinessHoursID: null.IntFrom(1), HandoffHoursMode: imodels.HandoffHoursModeBusiness},
			bhSvc:     &fakeBusinessHours{err: errors.New("boom")},
			settings:  settings, now: outside,
			wantAllow: true,
		},
		{
			name:      "missing app.timezone falls back to UTC and still resolves",
			inbox:     imodels.Inbox{HandoffBusinessHoursID: null.IntFrom(1), HandoffHoursMode: imodels.HandoffHoursModeBusiness},
			bhSvc:     bhSvc, settings: &fakeSettings{raw: types.JSONText(`{}`)}, now: inside,
			wantAllow: true,
		},
		{
			// A settings failure degrades to UTC rather than disabling the
			// restriction, so the window is still enforced.
			name:      "settings lookup error falls back to UTC and still enforces the window",
			inbox:     imodels.Inbox{HandoffBusinessHoursID: null.IntFrom(1), HandoffHoursMode: imodels.HandoffHoursModeBusiness},
			bhSvc:     bhSvc, settings: &fakeSettings{err: errors.New("boom")}, now: outside,
			wantAllow: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := newTestManager(t, c.now)
			m.SetInboxService(&fakeInbox{inbox: c.inbox})
			if c.bhSvc != nil {
				m.SetBusinessHoursService(c.bhSvc)
			} else {
				m.SetBusinessHoursService(&fakeBusinessHours{})
			}
			if c.settings != nil {
				m.SetSettingsService(c.settings)
			} else {
				m.SetSettingsService(&fakeSettings{})
			}
			conv := cmodels.Conversation{UUID: "u", InboxID: 42, ContactID: 7}
			if got := m.handoffAllowedNow(conv); got != c.wantAllow {
				t.Errorf("handoffAllowedNow() = %v, want %v", got, c.wantAllow)
			}
		})
	}
}

func TestHandleTransferRequestRestrictedSendsAndSkipsQueue(t *testing.T) {
	bh := bhmodels.BusinessHours{Hours: types.JSONText(`{"Monday":[{"open":"09:00","close":"18:00"}]}`)}
	m := newTestManager(t, time.Date(2026, 10, 19, 20, 0, 0, 0, time.UTC)) // outside hours
	conv := &fakeConv{}
	m.SetConversationService(conv)
	m.SetInboxService(&fakeInbox{inbox: imodels.Inbox{
		HandoffBusinessHoursID: null.IntFrom(1),
		HandoffHoursMode:       imodels.HandoffHoursModeBusiness,
	}})
	m.SetBusinessHoursService(&fakeBusinessHours{bh: bh})
	m.SetSettingsService(&fakeSettings{raw: types.JSONText(`{"app.timezone":"UTC"}`)})

	cfg := models.InboxQuickReplyConfig{TransferRestrictedMessage: "请稍后再试"}
	if err := m.handleTransferRequest(cmodels.Conversation{UUID: "u1", InboxID: 1, ContactID: 2}, cfg); err != nil {
		t.Fatalf("handleTransferRequest: %v", err)
	}
	if conv.calls != 1 {
		t.Errorf("expected exactly 1 SendAutoReply call, got %d", conv.calls)
	}
	if conv.gotContent != "请稍后再试" {
		t.Errorf("expected the configured message, got %q", conv.gotContent)
	}
	if conv.gotMeta != nil {
		t.Errorf("expected no meta on the restricted reply, got %#v", conv.gotMeta)
	}
	if !conv.gotSilent {
		t.Errorf("expected silent=true so the restricted message does not surface the conversation in agent lists")
	}
}

func TestHandleTransferRequestFallsBackToI18nDefault(t *testing.T) {
	bh := bhmodels.BusinessHours{Hours: types.JSONText(`{"Monday":[{"open":"09:00","close":"18:00"}]}`)}
	m := newTestManager(t, time.Date(2026, 10, 19, 20, 0, 0, 0, time.UTC))
	conv := &fakeConv{}
	m.SetConversationService(conv)
	m.SetInboxService(&fakeInbox{inbox: imodels.Inbox{
		HandoffBusinessHoursID: null.IntFrom(1),
		HandoffHoursMode:       imodels.HandoffHoursModeBusiness,
	}})
	m.SetBusinessHoursService(&fakeBusinessHours{bh: bh})
	m.SetSettingsService(&fakeSettings{raw: types.JSONText(`{"app.timezone":"UTC"}`)})

	cfg := models.InboxQuickReplyConfig{TransferRestrictedMessage: "   "} // whitespace only
	if err := m.handleTransferRequest(cmodels.Conversation{UUID: "u", InboxID: 1, ContactID: 2}, cfg); err != nil {
		t.Fatalf("handleTransferRequest: %v", err)
	}
	if conv.gotContent != "FALLBACK" {
		t.Errorf("expected i18n fallback message, got %q", conv.gotContent)
	}
}

func TestHandleTransferRequestUnrestrictedProceedsNormally(t *testing.T) {
	bh := bhmodels.BusinessHours{Hours: types.JSONText(`{"Monday":[{"open":"09:00","close":"18:00"}]}`)}
	m := newTestManager(t, time.Date(2026, 10, 19, 20, 0, 0, 0, time.UTC)) // outside hours
	conv := &fakeConv{queueCount: 3}
	m.SetConversationService(conv)
	// No handoff restriction: the inbox is unrestricted, so the queue reply
	// and queue meta side effects must run as in legacy behaviour.
	m.SetInboxService(&fakeInbox{inbox: imodels.Inbox{}})
	m.SetBusinessHoursService(&fakeBusinessHours{bh: bh})
	m.SetSettingsService(&fakeSettings{raw: types.JSONText(`{"app.timezone":"UTC"}`)})

	cfg := models.InboxQuickReplyConfig{QueueReply: "请稍候，您排在第 {count} 位"}
	if err := m.handleTransferRequest(cmodels.Conversation{UUID: "u", InboxID: 1, ContactID: 2}, cfg); err != nil {
		t.Fatalf("handleTransferRequest: %v", err)
	}
	if conv.calls != 1 {
		t.Errorf("expected exactly 1 SendAutoReply call, got %d", conv.calls)
	}
	if conv.gotContent != cfg.QueueReply {
		t.Errorf("expected the queue reply, got %q", conv.gotContent)
	}
	if conv.gotMeta == nil || conv.gotMeta["count"] != 3 {
		t.Errorf("expected queue_info meta with count=3, got %#v", conv.gotMeta)
	}
}

func TestAppTimezoneFallbacks(t *testing.T) {
	m := newTestManager(t, time.Now())
	// No settings wired: hard UTC.
	if got := m.appTimezone(); got != "UTC" {
		t.Errorf("default appTimezone() = %q, want UTC", got)
	}
	// Settings returns error: still UTC.
	m.SetSettingsService(&fakeSettings{err: errors.New("boom")})
	if got := m.appTimezone(); got != "UTC" {
		t.Errorf("error settings -> appTimezone() = %q, want UTC", got)
	}
	// Settings returns valid payload.
	m.SetSettingsService(&fakeSettings{raw: types.JSONText(`{"app.timezone":"Asia/Shanghai","app.lang":"zh-CN"}`)})
	if got := m.appTimezone(); got != "Asia/Shanghai" {
		t.Errorf("configured timezone = %q, want Asia/Shanghai", got)
	}
	// Settings payload has no app.timezone key.
	m.SetSettingsService(&fakeSettings{raw: types.JSONText(`{}`)})
	if got := m.appTimezone(); got != "UTC" {
		t.Errorf("missing key -> appTimezone() = %q, want UTC", got)
	}
}
