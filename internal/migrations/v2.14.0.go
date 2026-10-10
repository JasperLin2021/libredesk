package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_14_0 creates the tables backing the "客服知识库" (knowledge base) feature:
//   - `kb_categories` is a self-referencing, two-level category tree. Level-1
//     rows (parent_id IS NULL) are the "知识库" groups; level-2 rows belong to a
//     level-1 parent. The same row can also be uniquely identified by an
//     optional `code` so imported knowledge can reference categories by their
//     spreadsheet code.
//   - `kb_articles` holds the actual knowledge entries. They can be in either
//     the 'published' or 'draft' state. Drafts are surfaced in the UI under a
//     virtual "草稿" node filtered by `status='draft'`. `external_id` stores
//     the spreadsheet "知识ID" used by the bulk importer for upsert semantics.
//   - `kb_article_attachments` links an article to one or more media records
//     stored in the existing `media` table (uploads are sent through
//     `POST /api/v1/media` and referenced by UUID).
//
// The migration also grants the new `knowledge_base:manage` permission to the
// default Admin role so out-of-the-box installs can manage the knowledge base
// without a manual role configuration step.
func V2_14_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS kb_categories (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			parent_id BIGINT REFERENCES kb_categories(id) ON DELETE CASCADE ON UPDATE CASCADE,
			code TEXT,
			name TEXT NOT NULL,
			sort_order INT DEFAULT 0 NOT NULL,
			CONSTRAINT constraint_kb_categories_on_name CHECK (length(name) <= 50)
		);
		CREATE INDEX IF NOT EXISTS index_kb_categories_on_parent_id ON kb_categories (parent_id);
		CREATE UNIQUE INDEX IF NOT EXISTS index_uniq_kb_categories_on_parent_and_name
			ON kb_categories (COALESCE(parent_id, 0), name);
		CREATE UNIQUE INDEX IF NOT EXISTS index_uniq_kb_categories_on_parent_and_code
			ON kb_categories (COALESCE(parent_id, 0), code)
			WHERE code IS NOT NULL;
	`); err != nil {
		return err
	}

	// Grant the new permission to the default Admin role on existing installs.
	// Idempotent: the array_append only runs when the permission is missing.
	if _, err := db.Exec(`
		UPDATE roles
		SET permissions = array_append(permissions, 'knowledge_base:manage')
		WHERE name = 'Admin'
		  AND NOT ('knowledge_base:manage' = ANY(permissions));
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS kb_articles (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			category_id BIGINT REFERENCES kb_categories(id) ON DELETE SET NULL ON UPDATE CASCADE,
			external_id TEXT,
			code TEXT,
			title TEXT NOT NULL,
			author TEXT,
			content TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'published',
			sort_order INT DEFAULT 0 NOT NULL,
			CONSTRAINT constraint_kb_articles_on_title CHECK (length(title) <= 200),
			CONSTRAINT constraint_kb_articles_on_status CHECK (status IN ('published', 'draft'))
		);
		CREATE INDEX IF NOT EXISTS index_kb_articles_on_category_id ON kb_articles (category_id);
		CREATE INDEX IF NOT EXISTS index_kb_articles_on_status ON kb_articles (status);
		CREATE INDEX IF NOT EXISTS index_kb_articles_on_created_at ON kb_articles (created_at DESC);
		CREATE UNIQUE INDEX IF NOT EXISTS index_uniq_kb_articles_on_external_id
			ON kb_articles (external_id) WHERE external_id IS NOT NULL;
		CREATE UNIQUE INDEX IF NOT EXISTS index_uniq_kb_articles_on_code
			ON kb_articles (code) WHERE code IS NOT NULL;
	`); err != nil {
		return err
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS kb_article_attachments (
			id BIGSERIAL PRIMARY KEY,
			article_id BIGINT REFERENCES kb_articles(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
			media_uuid TEXT NOT NULL,
			file_name TEXT,
			content_type TEXT,
			size BIGINT,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS index_kb_article_attachments_on_article_id
			ON kb_article_attachments (article_id);
	`); err != nil {
		return err
	}

	return nil
}
