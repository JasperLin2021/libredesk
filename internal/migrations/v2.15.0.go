package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_15_0 creates the tables backing the "在线黑名单" (online blacklist)
// feature:
//   - `contact_blacklist` holds the current blacklist state. A contact is
//     blocked for as long as a row exists for them, which keeps the "is this
//     contact blocked?" lookup on the widget hot path a single indexed read.
//     `reason` and `created_by` back the 操作原因 / 操作人 columns of the
//     customer list.
//   - `contact_blacklist_logs` is the append-only history of every add/remove
//     operation and backs the "操作日志" page.
//
// The feature is deliberately separate from the pre-existing contact block
// (`PUT /api/v1/contacts/{id}/block`, `users.enabled = false`) so email
// channel behaviour is untouched.
func V2_15_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS contact_blacklist (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			contact_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
			reason TEXT NOT NULL DEFAULT '',
			created_by BIGINT REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE,
			CONSTRAINT constraint_contact_blacklist_on_reason CHECK (length(reason) <= 500)
		);
		CREATE UNIQUE INDEX IF NOT EXISTS index_uniq_contact_blacklist_on_contact_id
			ON contact_blacklist (contact_id);
		CREATE INDEX IF NOT EXISTS index_contact_blacklist_on_created_at
			ON contact_blacklist (created_at DESC);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS contact_blacklist_logs (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			contact_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
			action TEXT NOT NULL DEFAULT 'add',
			reason TEXT NOT NULL DEFAULT '',
			actor_id BIGINT REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE,
			CONSTRAINT constraint_contact_blacklist_logs_on_action CHECK (action IN ('add', 'remove')),
			CONSTRAINT constraint_contact_blacklist_logs_on_reason CHECK (length(reason) <= 500)
		);
		CREATE INDEX IF NOT EXISTS index_contact_blacklist_logs_on_contact_id
			ON contact_blacklist_logs (contact_id);
		CREATE INDEX IF NOT EXISTS index_contact_blacklist_logs_on_created_at
			ON contact_blacklist_logs (created_at DESC);
		CREATE INDEX IF NOT EXISTS index_contact_blacklist_logs_on_action
			ON contact_blacklist_logs (action);
	`); err != nil {
		return err
	}

	return nil
}
