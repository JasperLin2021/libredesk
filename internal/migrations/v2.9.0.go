package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_9_0 adds the `visitor_token` column to users table for persisting
// visitor tokens in the database as a fallback when Redis session is lost.
func V2_9_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		ALTER TABLE users
		ADD COLUMN IF NOT EXISTS visitor_token TEXT NULL;
	`)
	if err != nil {
		return err
	}

	// Add index for fast lookup by visitor token.
	_, err = db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS index_users_on_visitor_token
		ON users (visitor_token) WHERE visitor_token IS NOT NULL;
	`)
	return err
}
