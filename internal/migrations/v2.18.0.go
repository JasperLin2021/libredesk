package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_18_0 adds the `transfer_restricted_message` column to
// inbox_quick_reply_configs. It is sent automatically to the visitor when they
// ask to be transferred to a human agent outside the time window allowed by the
// inbox's "转人工设置" (human handoff) configuration.
//
// Existing rows keep NULL until the config is saved again; the read query
// COALESCEs the column so config rows written before this migration are
// unaffected.
func V2_18_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		ALTER TABLE inbox_quick_reply_configs
		ADD COLUMN IF NOT EXISTS transfer_restricted_message TEXT NULL;
	`)
	return err
}
