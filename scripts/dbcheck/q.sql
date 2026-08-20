SET search_path TO public;
SELECT id, type, sender_type, sender_id, left(content, 50) AS content, created_at
FROM conversation_messages
WHERE conversation_id = (SELECT id FROM conversations WHERE uuid = '81834899-7e18-4db5-9104-2abf85adedfb')
ORDER BY id LIMIT 30;
