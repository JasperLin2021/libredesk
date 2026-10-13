-- name: search-conversations-by-reference-number
-- Look up a conversation by exact reference number match. Optional filters
-- (date range, inbox, agent) are applied via positional sentinel parameters:
--   $2 = start time (nullable timestamptz)
--   $3 = end time   (nullable timestamptz)
--   $4 = inbox id array (nullable int[]); empty/NULL means no filter
--   $5 = assigned user id; 0 means no filter
SELECT
    conversations.created_at,
    conversations.uuid,
    conversations.reference_number,
    conversations.subject,
    cs.name AS status
FROM conversations
LEFT JOIN conversation_statuses cs ON conversations.status_id = cs.id
WHERE reference_number::text = $1
  AND ($2::timestamptz IS NULL OR conversations.created_at >= $2)
  AND ($3::timestamptz IS NULL OR conversations.created_at <= $3)
  AND ($4::int[] IS NULL OR cardinality($4) = 0 OR conversations.inbox_id = ANY($4))
  AND ($5::int <= 0 OR conversations.assigned_user_id = $5)
ORDER BY conversations.created_at DESC
LIMIT 1000;

-- name: search-conversations-by-contact-email
-- Search conversations whose contact email matches the query. Same optional
-- filters as `search-conversations-by-reference-number` ($2..$5).
SELECT
    conversations.created_at,
    conversations.uuid,
    conversations.reference_number,
    conversations.subject,
    cs.name AS status
FROM conversations
JOIN users ON conversations.contact_id = users.id
LEFT JOIN conversation_statuses cs ON conversations.status_id = cs.id
WHERE users.email ILIKE '%' || $1 || '%'
  AND ($2::timestamptz IS NULL OR conversations.created_at >= $2)
  AND ($3::timestamptz IS NULL OR conversations.created_at <= $3)
  AND ($4::int[] IS NULL OR cardinality($4) = 0 OR conversations.inbox_id = ANY($4))
  AND ($5::int <= 0 OR conversations.assigned_user_id = $5)
ORDER BY conversations.created_at DESC
LIMIT 1000;

-- name: search-messages
-- Search conversation messages by text content. Optional filters:
--   $2 = start time, $3 = end time (timestamptz) – applied to the
--        conversation's created_at so the period filter matches the design
--        ("时间排序"/"时间段" anchored on the conversation).
--   $4 = inbox id array (nullable int[])
--   $5 = assigned user id (0 = no filter)
SELECT
    m.uuid as "message_uuid",
    c.created_at as "conversation_created_at",
    c.reference_number as "conversation_reference_number",
    c.uuid as "conversation_uuid",
    LEFT(m.text_content, 200) AS text_content,
    cs.name as "conversation_status"
FROM conversation_messages m
    JOIN conversations c ON m.conversation_id = c.id
    LEFT JOIN conversation_statuses cs ON c.status_id = cs.id
WHERE m.type != 'activity' and m.text_content ILIKE '%' || $1 || '%'
  AND ($2::timestamptz IS NULL OR c.created_at >= $2)
  AND ($3::timestamptz IS NULL OR c.created_at <= $3)
  AND ($4::int[] IS NULL OR cardinality($4) = 0 OR c.inbox_id = ANY($4))
  AND ($5::int <= 0 OR c.assigned_user_id = $5)
ORDER BY c.created_at DESC, m.created_at DESC
LIMIT 30;

-- name: search-contacts
SELECT
    id,
    created_at,
    first_name,
    last_name,
    email,
    external_user_id
FROM users
WHERE type = 'contact'
AND deleted_at IS NULL
AND email ILIKE '%' || $1 || '%'
LIMIT 15;
