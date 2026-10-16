package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_17_0 adds the "转人工设置" (human handoff) configuration to inboxes.
// A livechat inbox can restrict when visitors are handed off to a human agent
// by picking a business hours to follow and a mode:
//   - `handoff_hours_mode = 'business'`: hand off only during the selected
//     working hours.
//   - `handoff_hours_mode = 'off_business'`: hand off only outside the selected
//     working hours (the inverse).
//
// A NULL `handoff_business_hours_id` means the handoff time is unrestricted and
// the mode is irrelevant. This keeps existing inboxes unaffected.
func V2_17_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		ALTER TABLE inboxes ADD COLUMN IF NOT EXISTS handoff_business_hours_id INT NULL REFERENCES business_hours(id) ON DELETE SET NULL ON UPDATE CASCADE;
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		ALTER TABLE inboxes ADD COLUMN IF NOT EXISTS handoff_hours_mode TEXT NOT NULL DEFAULT 'business';
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		ALTER TABLE inboxes DROP CONSTRAINT IF EXISTS constraint_inboxes_on_handoff_hours_mode;
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		ALTER TABLE inboxes ADD CONSTRAINT constraint_inboxes_on_handoff_hours_mode
			CHECK (handoff_hours_mode IN ('business', 'off_business'));
	`); err != nil {
		return err
	}

	return nil
}
