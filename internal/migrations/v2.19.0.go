package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_19_0 adds the `keyword_match_rules` table used by the "关键字匹配"
// (keyword match) feature: a global list of rules that are matched against
// visitor / agent messages and trigger an automatic reply or an agent-only
// system alert.
//
// It also grants the new `keyword_match:manage` permission to the built-in
// Admin role. Existing Admin rows keep their current permissions when they
// already carry the new permission, so re-runs are a no-op.
func V2_19_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS keyword_match_rules (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			name TEXT NOT NULL,
			source TEXT NOT NULL DEFAULT 'visitor',
			match_type TEXT NOT NULL DEFAULT 'keyword',
			pattern TEXT NOT NULL,
			case_sensitive BOOL DEFAULT FALSE NOT NULL,
			search_first BOOL DEFAULT FALSE NOT NULL,
			action TEXT NOT NULL DEFAULT 'agent_notify',
			reply_content TEXT NOT NULL DEFAULT '',
			enabled BOOL DEFAULT TRUE NOT NULL,
			sort_order INT DEFAULT 0 NOT NULL,
			CONSTRAINT constraint_keyword_match_rules_on_name CHECK (length(name) <= 300),
			CONSTRAINT constraint_keyword_match_rules_on_source CHECK (source IN ('visitor', 'agent')),
			CONSTRAINT constraint_keyword_match_rules_on_match_type CHECK (match_type IN ('keyword', 'regex')),
			CONSTRAINT constraint_keyword_match_rules_on_action CHECK (action IN ('agent_notify', 'agent_and_visitor'))
		);
		CREATE INDEX IF NOT EXISTS index_keyword_match_rules_on_enabled_and_sort_order
			ON keyword_match_rules (enabled, sort_order);
	`); err != nil {
		return err
	}

	// Grant the new permission to the built-in Admin role.
	_, err := db.Exec(`
		UPDATE roles
		SET permissions = array_append(permissions, 'keyword_match:manage')
		WHERE name = 'Admin'
			AND NOT ('keyword_match:manage' = ANY(permissions));
	`)
	return err
}
