-- name: get-blacklist
SELECT
    id,
    contact_id,
    reason,
    created_by,
    created_at,
    updated_at
FROM
    contact_blacklist
WHERE
    contact_id = $1;

-- name: upsert-blacklist
-- A contact can only be blacklisted once; re-adding updates the reason and
-- the acting agent so the latest operation wins.
INSERT INTO
    contact_blacklist (contact_id, reason, created_by)
VALUES
    ($1, $2, $3)
ON CONFLICT (contact_id) DO UPDATE
SET
    reason = EXCLUDED.reason,
    created_by = EXCLUDED.created_by,
    updated_at = NOW()
RETURNING
    id,
    contact_id,
    reason,
    created_by,
    created_at,
    updated_at;

-- name: delete-blacklist
DELETE FROM
    contact_blacklist
WHERE
    contact_id = $1;

-- name: insert-blacklist-log
INSERT INTO
    contact_blacklist_logs (contact_id, action, reason, actor_id)
VALUES
    ($1, $2, $3, $4)
RETURNING
    id;

-- name: count-blacklisted
SELECT
    COUNT(*)::INT AS total
FROM
    contact_blacklist;

-- name: get-blacklist-customers
-- Customer list backing "服务历史 > 客户 > 客户信息". `filter` selects the tab
-- ('all' lists every contact, 'blocked' only those currently blacklisted).
-- The ORDER BY / LIMIT / OFFSET clauses are appended by the manager so the
-- sort column can be picked safely from a whitelist.
SELECT
    COUNT(*) OVER() AS total,
    u.id AS contact_id,
    COALESCE(
        NULLIF(TRIM(CONCAT(u.first_name, ' ', u.last_name)), ''),
        NULLIF(u.email, ''),
        '#' || u.id
    ) AS contact_name,
    u.type AS contact_type,
    u.created_at AS contact_created_at,
    COALESCE(u.external_user_id, '') AS external_user_id,
    COALESCE(u.phone_number, '') AS phone_number,
    COALESCE(u.phone_number_country_code, '') AS phone_number_country_code,
    COALESCE(u.email, '') AS contact_email,
    (b.contact_id IS NOT NULL) AS blocked,
    COALESCE(b.reason, '') AS reason,
    b.created_at AS blacklisted_at,
    COALESCE(b.created_by, 0) AS actor_id,
    COALESCE(
        NULLIF(TRIM(CONCAT(a.first_name, ' ', a.last_name)), ''),
        NULLIF(a.email, ''),
        ''
    ) AS actor_name
FROM
    users u
    LEFT JOIN contact_blacklist b ON b.contact_id = u.id
    LEFT JOIN users a ON a.id = b.created_by
WHERE
    u.deleted_at IS NULL
    AND u.type IN ('contact', 'visitor')
    AND ($1::text <> 'blocked' OR b.contact_id IS NOT NULL)
    AND ($2::timestamptz IS NULL OR b.created_at >= $2)
    AND ($3::timestamptz IS NULL OR b.created_at < $3)
    AND (
        $4::text = ''
        OR COALESCE(
            NULLIF(TRIM(CONCAT(u.first_name, ' ', u.last_name)), ''),
            NULLIF(u.email, ''),
            ''
        ) ILIKE '%' || $4 || '%'
        OR COALESCE(
            NULLIF(TRIM(CONCAT(a.first_name, ' ', a.last_name)), ''),
            NULLIF(a.email, ''),
            ''
        ) ILIKE '%' || $4 || '%'
        OR COALESCE(b.reason, '') ILIKE '%' || $4 || '%'
    )

-- name: get-blacklist-logs
-- Append-only operation log backing "服务历史 > 客户 > 操作日志".
-- The ORDER BY / LIMIT / OFFSET clauses are appended by the manager.
SELECT
    COUNT(*) OVER() AS total,
    l.id,
    l.contact_id,
    COALESCE(
        NULLIF(TRIM(CONCAT(c.first_name, ' ', c.last_name)), ''),
        NULLIF(c.email, ''),
        '#' || l.contact_id
    ) AS contact_name,
    l.action,
    COALESCE(l.reason, '') AS reason,
    COALESCE(l.actor_id, 0) AS actor_id,
    COALESCE(
        NULLIF(TRIM(CONCAT(a.first_name, ' ', a.last_name)), ''),
        NULLIF(a.email, ''),
        ''
    ) AS actor_name,
    l.created_at
FROM
    contact_blacklist_logs l
    LEFT JOIN users c ON c.id = l.contact_id
    LEFT JOIN users a ON a.id = l.actor_id
WHERE
    ($1::text = '' OR l.action = $1)
    AND ($2::timestamptz IS NULL OR l.created_at >= $2)
    AND ($3::timestamptz IS NULL OR l.created_at < $3)
    AND (
        $4::text = ''
        OR COALESCE(
            NULLIF(TRIM(CONCAT(c.first_name, ' ', c.last_name)), ''),
            NULLIF(c.email, ''),
            ''
        ) ILIKE '%' || $4 || '%'
        OR COALESCE(
            NULLIF(TRIM(CONCAT(a.first_name, ' ', a.last_name)), ''),
            NULLIF(a.email, ''),
            ''
        ) ILIKE '%' || $4 || '%'
        OR COALESCE(l.reason, '') ILIKE '%' || $4 || '%'
    )
