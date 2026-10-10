package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_13_0 creates the per-user tables backing the agent-facing "常用语"
// sidebar tab:
//   - `user_canned_responses` holds the personal canned responses authored
//     by each agent. They are isolated by `user_id` so agents only ever see
//     and operate on their own entries.
//   - `user_canned_pins` records which canned responses an agent has pinned
//     to the top of the sidebar. The same table is used for both globally
//     shared ("公共") responses and personal ("个人") responses; the `source`
//     column discriminates between them and `response_id` is resolved against
//     `canned_responses` or `user_canned_responses` at read time.
func V2_13_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS user_canned_responses (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			user_id BIGINT REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
			content TEXT NOT NULL,
			sort_order INT DEFAULT 0 NOT NULL,
			CONSTRAINT constraint_user_canned_responses_on_content CHECK (length(content) <= 1000)
		);
		CREATE INDEX IF NOT EXISTS index_user_canned_responses_on_user_id ON user_canned_responses (user_id);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS user_canned_pins (
			user_id BIGINT REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
			source TEXT NOT NULL,
			response_id BIGINT NOT NULL,
			sort_order INT DEFAULT 0 NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			PRIMARY KEY (user_id, source, response_id),
			CONSTRAINT constraint_user_canned_pins_on_source CHECK (source IN ('global', 'personal'))
		);
		CREATE INDEX IF NOT EXISTS index_user_canned_pins_on_user_id ON user_canned_pins (user_id);
	`); err != nil {
		return err
	}

	return nil
}
