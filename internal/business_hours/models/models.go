// Package models contains the data models for the businesshours package.
package models

import (
	"encoding/json"
	"time"

	"github.com/jmoiron/sqlx/types"
	"github.com/volatiletech/null/v9"
)

// BusinessHours represents the business in the database.
type BusinessHours struct {
	ID           int            `db:"id" json:"id"`
	CreatedAt    time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time      `db:"updated_at" json:"updated_at"`
	Name         string         `db:"name" json:"name"`
	Description  null.String    `db:"description" json:"description"`
	IsAlwaysOpen bool           `db:"is_always_open" json:"is_always_open"`
	Holidays     types.JSONText `db:"holidays" json:"holidays"`
	Hours        types.JSONText `db:"hours" json:"hours"`
}

// WorkingHours represents a single working time range within a day.
// A day may have multiple ranges, stored as a JSON array.
type WorkingHours struct {
	Open  string `json:"open"`
	Close string `json:"close"`
}

// Holiday represents a holiday spanning a date range (inclusive).
type Holiday struct {
	Name      string `json:"name"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

// UnmarshalJSON supports the legacy single-day format
// {"name": "...", "date": "2024-01-01"} by mapping "date" to both
// StartDate and EndDate.
func (h *Holiday) UnmarshalJSON(b []byte) error {
	type alias Holiday
	var a alias
	if err := json.Unmarshal(b, &a); err != nil {
		return err
	}
	*h = Holiday(a)
	if h.StartDate == "" {
		var legacy struct {
			Date string `json:"date"`
		}
		if err := json.Unmarshal(b, &legacy); err == nil && legacy.Date != "" {
			h.StartDate = legacy.Date
			h.EndDate = legacy.Date
		}
	}
	return nil
}
