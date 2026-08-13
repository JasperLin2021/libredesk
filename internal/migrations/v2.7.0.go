package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_7_0 adds the quick reply (automatic reply) configuration and content
// tables used by the per-inbox guided conversation / transfer to human flow.
func V2_7_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS inbox_quick_reply_configs (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			inbox_id INT REFERENCES inboxes(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
			welcome_message TEXT NULL,
			transfer_keyword TEXT NOT NULL DEFAULT '我要转人工',
			queue_reply TEXT NULL,
			assigned_reply TEXT NULL,
			closed_reply TEXT NULL,
			enabled BOOL DEFAULT FALSE NOT NULL,
			CONSTRAINT constraint_inbox_quick_reply_configs_on_inbox_id UNIQUE (inbox_id)
		);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS quick_reply_topics (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			inbox_id INT REFERENCES inboxes(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
			name TEXT NOT NULL,
			sort_order INT DEFAULT 0 NOT NULL,
			CONSTRAINT constraint_quick_reply_topics_on_inbox_id_and_name UNIQUE (inbox_id, name)
		);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS index_quick_reply_topics_on_inbox_id ON quick_reply_topics (inbox_id);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS quick_reply_questions (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			topic_id BIGINT REFERENCES quick_reply_topics(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
			question TEXT NOT NULL,
			answer TEXT NULL,
			sort_order INT DEFAULT 0 NOT NULL
		);
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS index_quick_reply_questions_on_topic_id ON quick_reply_questions (topic_id);
	`); err != nil {
		return err
	}

	return nil
}
