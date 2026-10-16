package sla

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"time"

	"github.com/abhinavxd/libredesk/internal/business_hours/models"
)

var (
	ErrInvalidSLADuration = fmt.Errorf("invalid SLA duration")
	ErrMaxIterations      = fmt.Errorf("sla: exceeded maximum iterations - check configuration")
	ErrInvalidTime        = fmt.Errorf("invalid time")
)

// CalculateDeadline computes the SLA deadline from a start time and SLA duration in minutes
// considering the provided holidays, working hours, and time zone.
func (m *Manager) CalculateDeadline(start time.Time, slaMinutes int, businessHours models.BusinessHours, timeZone string) (time.Time, error) {
	if slaMinutes <= 0 {
		return time.Time{}, ErrInvalidSLADuration
	}

	// If business is always open, return the deadline as the start time plus the SLA duration.
	if businessHours.IsAlwaysOpen {
		return start.Add(time.Duration(slaMinutes) * time.Minute), nil
	}

	// Load the specified time zone.
	loc, err := time.LoadLocation(timeZone)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid time zone %s: %v", timeZone, err)
	}

	// Convert start time to the specified time zone.
	currentTime := start.In(loc)
	remainingMinutes := slaMinutes
	maxIterations := ((slaMinutes+59)/60)*24 + 1

	// Unmarshal working hours. Supports both the current array format
	// {"Monday": [{"open": "09:00", "close": "12:00"}, {"open": "14:00", "close": "18:00"}]}
	// and the legacy single-range format {"Monday": {"open": "09:00", "close": "17:00"}}.
	var rawHours map[string]json.RawMessage
	if err := json.Unmarshal(businessHours.Hours, &rawHours); err != nil {
		return time.Time{}, fmt.Errorf("could not unmarshal working hours for SLA deadline calcuation: %v", err)
	}

	workingHours := make(map[string][]models.WorkingHours, len(rawHours))
	for day, raw := range rawHours {
		var ranges []models.WorkingHours
		if err := json.Unmarshal(raw, &ranges); err != nil {
			var single models.WorkingHours
			if err := json.Unmarshal(raw, &single); err != nil {
				return time.Time{}, fmt.Errorf("could not unmarshal working hours for %s: %v", day, err)
			}
			ranges = []models.WorkingHours{single}
		}
		// Sort ranges by open time so they are consumed in order.
		sort.Slice(ranges, func(i, j int) bool { return ranges[i].Open < ranges[j].Open })
		workingHours[day] = ranges
	}

	// Unmarshal holidays.
	var holidays = []models.Holiday{}
	if len(businessHours.Holidays) > 0 {
		if err := json.Unmarshal(businessHours.Holidays, &holidays); err != nil {
			return time.Time{}, fmt.Errorf("could not unmarshal holidays for SLA deadline calcuation: %v", err)
		}
	}

	// Create a map of holidays, expanding date ranges into individual dates.
	holidaysMap := make(map[string]struct{})
	for _, holiday := range holidays {
		holidayStart, err := time.Parse(time.DateOnly, holiday.StartDate)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid holiday start date %s: %v", holiday.StartDate, err)
		}
		holidayEnd, err := time.Parse(time.DateOnly, holiday.EndDate)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid holiday end date %s: %v", holiday.EndDate, err)
		}
		for d := holidayStart; !d.After(holidayEnd); d = d.AddDate(0, 0, 1) {
			holidaysMap[d.Format(time.DateOnly)] = struct{}{}
		}
	}

	iterations := 0
	for remainingMinutes > 0 {
		iterations++
		if iterations > maxIterations {
			return time.Time{}, ErrMaxIterations
		}

		// Move to next day if current day is a holiday.
		dateStr := currentTime.Format(time.DateOnly)
		if _, isHoliday := holidaysMap[dateStr]; isHoliday {
			currentTime = nextDay(currentTime, loc)
			continue
		}

		// Get working hours for the current day.
		dayOfWeek := currentTime.Weekday().String()
		dayRanges, exists := workingHours[dayOfWeek]

		// Not a working day, move to next day.
		if !exists || len(dayRanges) == 0 {
			currentTime = nextDay(currentTime, loc)
			continue
		}

		// Consume the working time ranges of the current day in order.
		for _, workHours := range dayRanges {
			// Parse open and close times for the current range in the specified time zone.
			startOfWork, err := parseTime(currentTime, workHours.Open, loc)
			if err != nil {
				return time.Time{}, fmt.Errorf("invalid open time %s for %s: %v", workHours.Open, dayOfWeek, err)
			}
			endOfWork, err := parseTime(currentTime, workHours.Close, loc)
			if err != nil {
				return time.Time{}, fmt.Errorf("invalid close time %s for %s: %v", workHours.Close, dayOfWeek, err)
			}

			// Current time is past this range, try the next range of the day.
			if !currentTime.Before(endOfWork) {
				continue
			}

			// Adjust to start of work if current time is before it.
			if currentTime.Before(startOfWork) {
				currentTime = startOfWork
			}

			// Deduct minutes worked in this range from remaining SLA time.
			workMinutesLeft := int(endOfWork.Sub(currentTime).Minutes())
			if workMinutesLeft >= remainingMinutes {
				return currentTime.Add(time.Duration(remainingMinutes) * time.Minute), nil
			}

			remainingMinutes -= workMinutesLeft
			if remainingMinutes == 0 {
				return endOfWork, nil
			}

			currentTime = endOfWork
		}

		// All ranges of the day are consumed, move to next day.
		currentTime = nextDay(currentTime, loc)
	}

	return currentTime, nil
}

// nextDay advances the time to the start of the next day in the specified time zone.
func nextDay(t time.Time, loc *time.Location) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, loc)
}

// parseTime parses a time string in "HH:MM" format and returns a time.Time object for the given date and location.
func parseTime(date time.Time, timeStr string, loc *time.Location) (time.Time, error) {
	// Validate timeStr is in "HH:MM" format.
	matched, err := regexp.MatchString(`^(?:[01]\d|2[0-3]):[0-5]\d$`, timeStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("could not validate time string %s: %v", timeStr, err)
	}
	if !matched {
		return time.Time{}, ErrInvalidTime
	}

	parsedTime, err := time.ParseInLocation("15:04", timeStr, loc)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(date.Year(), date.Month(), date.Day(), parsedTime.Hour(), parsedTime.Minute(), 0, 0, loc), nil
}
