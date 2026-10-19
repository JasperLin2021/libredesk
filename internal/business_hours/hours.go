package businesshours

import (
	"encoding/json"
	"errors"
	"sort"
	"time"

	bmodels "github.com/abhinavxd/libredesk/internal/business_hours/models"
)

// errInvalidWorkingHours is returned when the per-day working hours payload is
// neither an array of ranges nor a legacy single range object.
var errInvalidWorkingHours = errors.New("invalid business hours")

// IsWithinBusinessHours reports whether the given `now` (interpreted in the
// supplied IANA time zone, with UTC as the fallback for invalid zones) falls
// within the business hours schedule.
//
// Behaviour:
//   - `IsAlwaysOpen` short-circuits to true.
//   - Holidays declared as inclusive date ranges fully cover `now`'s day and
//     yield false.
//   - The working hours JSON supports the current array format
//     `{"Monday":[{"open":"09:00","close":"12:00"}, ...]}` and the legacy
//     single-range format `{"Monday":{"open":..,"close":..}}`.
//   - A day with no working ranges, or a schedule with no hours at all, is
//     treated as closed so a malformed schedule never silently grants 24/7
//     access. An individual malformed range is skipped rather than widening
//     access.
//
// Callers that only care about "is it open right now" should treat a non-nil
// error as unknown rather than as closed.
func (m *Manager) IsWithinBusinessHours(bh bmodels.BusinessHours, tz string, now time.Time) (bool, error) {
	if bh.IsAlwaysOpen {
		return true, nil
	}

	loc, err := time.LoadLocation(tz)
	if err != nil || loc == nil {
		loc = time.UTC
	}
	localNow := now.In(loc)

	// Holidays: expand inclusive date ranges and check today.
	holidays := []bmodels.Holiday{}
	if len(bh.Holidays) > 0 {
		if err := json.Unmarshal(bh.Holidays, &holidays); err != nil {
			m.lo.Error("error unmarshalling business hours holidays", "error", err)
			return false, err
		}
	}
	holidaySet := expandHolidays(holidays)
	dateStr := localNow.Format(time.DateOnly)
	if _, isHoliday := holidaySet[dateStr]; isHoliday {
		return false, nil
	}

	// Not operating at all when there are no working hours configured.
	if len(bh.Hours) == 0 {
		return false, nil
	}

	// Working hours: support array-of-ranges and legacy single-object formats.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(bh.Hours, &raw); err != nil {
		m.lo.Error("error unmarshalling business hours hours", "error", err)
		return false, err
	}
	dayRanges, ok := raw[localNow.Weekday().String()]
	if !ok || len(dayRanges) == 0 {
		return false, nil
	}
	ranges, err := decodeRanges(dayRanges)
	if err != nil {
		m.lo.Error("error unmarshalling business hours ranges", "day", localNow.Weekday().String(), "error", err)
		return false, nil
	}
	// Sort by open time so the day is consumed in order; this matches the SLA
	// calculator's behaviour and keeps the logic predictable.
	sort.Slice(ranges, func(i, j int) bool { return ranges[i].Open < ranges[j].Open })

	for _, r := range ranges {
		open, ok1 := parseClock(localNow, r.Open)
		close, ok2 := parseClock(localNow, r.Close)
		if !ok1 || !ok2 {
			// Skip malformed ranges so they do not poison the whole day.
			continue
		}
		if !localNow.Before(open) && localNow.Before(close) {
			return true, nil
		}
	}
	return false, nil
}

// decodeRanges parses the per-day working hours payload. It tries the array
// format first and falls back to a single WorkingHours object for legacy
// configs. An empty payload returns an empty slice with no error.
func decodeRanges(raw json.RawMessage) ([]bmodels.WorkingHours, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var ranges []bmodels.WorkingHours
	if err := json.Unmarshal(raw, &ranges); err == nil {
		return ranges, nil
	}
	var single bmodels.WorkingHours
	if err := json.Unmarshal(raw, &single); err == nil {
		return []bmodels.WorkingHours{single}, nil
	}
	return nil, errInvalidWorkingHours
}

// expandHolidays returns a set of "YYYY-MM-DD" strings covered by the inclusive
// date ranges in the given holidays. An invalid date in any holiday is skipped
// so it cannot widen or shrink the set unexpectedly.
func expandHolidays(holidays []bmodels.Holiday) map[string]struct{} {
	set := make(map[string]struct{})
	for _, h := range holidays {
		start, err := time.Parse(time.DateOnly, h.StartDate)
		if err != nil {
			continue
		}
		end, err := time.Parse(time.DateOnly, h.EndDate)
		if err != nil {
			continue
		}
		for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
			set[d.Format(time.DateOnly)] = struct{}{}
		}
	}
	return set
}

// parseClock returns the time of day (in the time zone of `day`) corresponding
// to the given "HH:MM" string. The second return is false if the string is
// malformed.
func parseClock(day time.Time, clock string) (time.Time, bool) {
	t, err := time.ParseInLocation("15:04", clock, day.Location())
	if err != nil {
		return time.Time{}, false
	}
	return time.Date(day.Year(), day.Month(), day.Day(), t.Hour(), t.Minute(), 0, 0, day.Location()), true
}
