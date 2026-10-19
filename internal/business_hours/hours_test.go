package businesshours

import (
	"testing"
	"time"

	bmodels "github.com/abhinavxd/libredesk/internal/business_hours/models"
	"github.com/jmoiron/sqlx/types"
	"github.com/zerodha/logf"
)

func newTestManager() *Manager {
	lo := logf.New(logf.Opts{Level: logf.DebugLevel})
	return &Manager{lo: &lo}
}

func TestIsWithinBusinessHours(t *testing.T) {
	m := newTestManager()
	// 2026-10-19 is a Monday. Use Asia/Shanghai for a non-UTC zone and assert
	// the schedule is interpreted in that zone, not the server's.
	bh := bmodels.BusinessHours{
		Hours:    types.JSONText(`{"Monday":[{"open":"09:00","close":"12:00"},{"open":"14:00","close":"18:00"}]}`),
		Holidays: types.JSONText(`[]`),
	}

	cases := []struct {
		name string
		now  string
		want bool
	}{
		{"before first range", "2026-10-19T00:00:00+08:00", false},
		{"first range open boundary", "2026-10-19T09:00:00+08:00", true},
		{"inside first range", "2026-10-19T11:59:00+08:00", true},
		{"first range close boundary", "2026-10-19T12:00:00+08:00", false},
		{"lunch break", "2026-10-19T13:00:00+08:00", false},
		{"second range open boundary", "2026-10-19T14:00:00+08:00", true},
		{"second range close boundary", "2026-10-19T18:00:00+08:00", false},
		{"after hours", "2026-10-19T23:00:00+08:00", false},
		{"different weekday with no ranges", "2026-10-20T10:00:00+08:00", false},
	}

	for _, c := range cases {
		now, err := time.Parse(time.RFC3339, c.now)
		if err != nil {
			t.Fatalf("failed to parse time %s: %v", c.now, err)
		}
		// Pass the instant in UTC to prove it is converted into the given tz.
		got, err := m.IsWithinBusinessHours(bh, "Asia/Shanghai", now.In(time.UTC))
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", c.name, err)
		}
		if got != c.want {
			t.Errorf("%s: IsWithinBusinessHours() = %v, want %v", c.name, got, c.want)
		}
	}

	// Sanity: the schedule must be resolved in Asia/Shanghai (UTC+8), not in
	// the UTC wall clock of the instant. 12:00 UTC is 20:00 in Shanghai,
	// i.e. after the 18:00 close.
	twelveUTC := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	if got, _ := m.IsWithinBusinessHours(bh, "Asia/Shanghai", twelveUTC); got {
		t.Errorf("expected 12:00 UTC (20:00 Asia/Shanghai) to be closed")
	}
}

func TestIsWithinBusinessHoursAlwaysOpen(t *testing.T) {
	m := newTestManager()
	bh := bmodels.BusinessHours{IsAlwaysOpen: true, Hours: types.JSONText(`{}`)}
	got, err := m.IsWithinBusinessHours(bh, "Asia/Shanghai", time.Now())
	if err != nil || !got {
		t.Errorf("IsWithinBusinessHours() = (%v, %v), want (true, nil)", got, err)
	}
}

func TestIsWithinBusinessHoursLegacyFormat(t *testing.T) {
	m := newTestManager()
	bh := bmodels.BusinessHours{Hours: types.JSONText(`{"Monday":{"open":"09:00","close":"17:00"}}`)}
	now := time.Date(2026, 10, 19, 10, 0, 0, 0, time.UTC)
	if got, err := m.IsWithinBusinessHours(bh, "UTC", now); err != nil || !got {
		t.Errorf("legacy format: IsWithinBusinessHours() = (%v, %v), want (true, nil)", got, err)
	}
	if got, err := m.IsWithinBusinessHours(bh, "UTC", now.Add(10*time.Hour)); err != nil || got {
		t.Errorf("legacy format after close: IsWithinBusinessHours() = (%v, %v), want (false, nil)", got, err)
	}
}

func TestIsWithinBusinessHoursHoliday(t *testing.T) {
	m := newTestManager()
	bh := bmodels.BusinessHours{
		Hours:    types.JSONText(`{"Monday":[{"open":"09:00","close":"17:00"}]}`),
		Holidays: types.JSONText(`[{"name":"Holiday","start_date":"2026-10-17","end_date":"2026-10-20"}]`),
	}
	// 2026-10-19 is inside the inclusive holiday range even though it is a
	// configured working day.
	now := time.Date(2026, 10, 19, 10, 0, 0, 0, time.UTC)
	if got, err := m.IsWithinBusinessHours(bh, "UTC", now); err != nil || got {
		t.Errorf("holiday: IsWithinBusinessHours() = (%v, %v), want (false, nil)", got, err)
	}
	// Monday falling outside the holiday range is a normal working day.
	bh.Holidays = types.JSONText(`[{"name":"Holiday","start_date":"2026-10-21","end_date":"2026-10-22"}]`)
	if got, err := m.IsWithinBusinessHours(bh, "UTC", now); err != nil || !got {
		t.Errorf("outside holiday: IsWithinBusinessHours() = (%v, %v), want (true, nil)", got, err)
	}
}

func TestIsWithinBusinessHoursInvalidInputs(t *testing.T) {
	m := newTestManager()
	// No hours configured at all must be closed, not open.
	bh := bmodels.BusinessHours{}
	if got, err := m.IsWithinBusinessHours(bh, "UTC", time.Now()); err != nil || got {
		t.Errorf("empty hours: IsWithinBusinessHours() = (%v, %v), want (false, nil)", got, err)
	}
	// Invalid time zone falls back to UTC instead of erroring.
	bh = bmodels.BusinessHours{Hours: types.JSONText(`{"Monday":[{"open":"09:00","close":"17:00"}]}`)}
	now := time.Date(2026, 10, 19, 10, 0, 0, 0, time.UTC)
	if got, err := m.IsWithinBusinessHours(bh, "Not/AZone", now); err != nil || !got {
		t.Errorf("invalid tz: IsWithinBusinessHours() = (%v, %v), want (true, nil)", got, err)
	}
	// Malformed hours payload surfaces an error.
	bh = bmodels.BusinessHours{Hours: types.JSONText(`not json`)}
	if _, err := m.IsWithinBusinessHours(bh, "UTC", now); err == nil {
		t.Errorf("malformed hours: expected an error, got nil")
	}
}
