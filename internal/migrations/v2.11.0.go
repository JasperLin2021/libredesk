package migrations

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_11_0 translates existing English activity message content into Chinese for
// deployments where the app language is set to zh-CN. Activity messages are
// stored as localized snapshots, so older messages (created before activity
// content was localized) keep their English text. This migration brings them in
// line with newly generated messages. The update is idempotent: already
// translated content does not match the English patterns and is left untouched,
// and the migration is a no-op for non-Chinese deployments.
func V2_11_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	// Resolve the configured app language, preferring the value stored in DB
	// settings (which can be changed from the admin panel) over the config file.
	lang := ko.String("app.lang")
	if err := db.QueryRowx(`SELECT value->>'lang' FROM settings WHERE key = 'app'`).Scan(&lang); err != nil && err != sql.ErrNoRows {
		return err
	}
	if lang != "zh-CN" {
		return nil
	}

	updates := []struct {
		activityType string
		pattern      string
		replacement  string
	}{
		{"assigned_user_change", `^Assigned to (.*?) by (.*)$`, `已分配给 \1（由 \2 操作）`},
		{"assigned_team_change", `^Assigned to (.*?) team by (.*)$`, `已分配给 \1 团队（由 \2 操作）`},
		{"self_assign", `^(.*?) self-assigned this conversation$`, `\1 自行认领了该会话`},
		{"priority_change", `^(.*?) set priority to (.*)$`, `\1 将优先级设置为 \2`},
		{"status_change", `^(.*?) marked the conversation as (.*)$`, `\1 将会话状态标记为 \2`},
		{"tag_added", `^(.*?) added tag (.*)$`, `\1 添加了标签 \2`},
		{"tag_removed", `^(.*?) removed tag (.*)$`, `\1 移除了标签 \2`},
		{"sla_set", `^(.*?) set (.*) SLA policy$`, `\1 设置了 \2 SLA 策略`},
		{"participant_added", `^(.*?) joined the conversation$`, `\1 加入了会话`},
	}

	for _, u := range updates {
		if _, err := db.Exec(`
			UPDATE messages
			SET content = regexp_replace(content, $1, $2)
			WHERE type = 'activity'
			  AND meta->>'activity_type' = $3
			  AND content ~ $1`, u.pattern, u.replacement, u.activityType); err != nil {
			return err
		}
	}
	return nil
}
