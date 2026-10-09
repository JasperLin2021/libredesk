package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_12_0 creates the tables backing the globally shared "canned responses"
// (公共常用语) feature: a self-referencing `canned_categories` table supporting
// arbitrarily nested categories, and a `canned_responses` table holding the
// canned response entries that belong to a category. Both are global (not
// scoped to an inbox), unlike the existing quick reply tables.
func V2_12_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS canned_categories (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			parent_id BIGINT REFERENCES canned_categories(id) ON DELETE CASCADE ON UPDATE CASCADE,
			name TEXT NOT NULL,
			sort_order INT DEFAULT 0 NOT NULL,
			CONSTRAINT constraint_canned_categories_on_name CHECK (length(name) <= 20)
		);
		CREATE INDEX IF NOT EXISTS index_canned_categories_on_parent_id ON canned_categories (parent_id);
		CREATE UNIQUE INDEX IF NOT EXISTS index_uniq_canned_categories_on_parent_and_name
			ON canned_categories (COALESCE(parent_id, 0), name);
	`); err != nil {
		return err
	}

	// Grant the new permission to the default Admin role on existing installs.
	// Idempotent: the array_append only runs when the permission is missing.
	if _, err := db.Exec(`
		UPDATE roles
		SET permissions = array_append(permissions, 'canned_responses:manage')
		WHERE name = 'Admin'
		  AND NOT ('canned_responses:manage' = ANY(permissions));
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS canned_responses (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			category_id BIGINT REFERENCES canned_categories(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
			content TEXT NOT NULL,
			sort_order INT DEFAULT 0 NOT NULL,
			CONSTRAINT constraint_canned_responses_on_content CHECK (length(content) <= 1000)
		);
		CREATE INDEX IF NOT EXISTS index_canned_responses_on_category_id ON canned_responses (category_id);
	`); err != nil {
		return err
	}

	return nil
}
