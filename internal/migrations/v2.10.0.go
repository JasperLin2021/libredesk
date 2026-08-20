package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_10_0 adds the `no_reply_timeout_reply` column to inbox_quick_reply_configs
// so that a preset message can be sent (as the assigned agent) when a visitor
// does not reply within the configured no-reply timeout before the conversation
// is auto-closed.
func V2_10_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		ALTER TABLE inbox_quick_reply_configs
		ADD COLUMN IF NOT EXISTS no_reply_timeout_reply TEXT NULL;
	`)
	return err
}
