package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_8_0 adds the `names` TEXT[] column to quick_reply_topics so that a single
// topic can have multiple display names (aliases) that all map to the same set
// of questions.
func V2_8_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	// Add the names column if it does not exist.
	if _, err := db.Exec(`
		ALTER TABLE quick_reply_topics
		ADD COLUMN IF NOT EXISTS names TEXT[] NOT NULL DEFAULT '{}'::TEXT[];
	`); err != nil {
		return err
	}

	// Back-fill: for existing rows populate names with {name}.
	if _, err := db.Exec(`
		UPDATE quick_reply_topics
		SET names = ARRAY[name]
		WHERE names = '{}'::TEXT[] OR names IS NULL;
	`); err != nil {
		return err
	}

	// Drop the old unique constraint on (inbox_id, name) since uniqueness is
	// now enforced across all aliases via application logic.
	if _, err := db.Exec(`
		ALTER TABLE quick_reply_topics
		DROP CONSTRAINT IF EXISTS constraint_quick_reply_topics_on_inbox_id_and_name;
	`); err != nil {
		return err
	}

	// Use a GIN index for efficient overlap/containment checks when matching
	// topics by any of their names.
	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS index_quick_reply_topics_on_names
		ON quick_reply_topics USING GIN (names);
	`); err != nil {
		return err
	}

	return nil
}
