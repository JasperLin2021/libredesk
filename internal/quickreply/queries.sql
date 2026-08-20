-- name: get-config
-- COALESCE the nullable TEXT columns so that legacy NULL values scan into
-- Go string fields without errors.
select
    id,
    inbox_id,
    COALESCE(welcome_message, '') AS welcome_message,
    transfer_keyword,
    COALESCE(queue_reply, '') AS queue_reply,
    COALESCE(assigned_reply, '') AS assigned_reply,
    COALESCE(closed_reply, '') AS closed_reply,
    COALESCE(no_reply_timeout_reply, '') AS no_reply_timeout_reply,
    enabled,
    created_at,
    updated_at
from
    inbox_quick_reply_configs
where
    inbox_id = $1;

-- name: upsert-config
insert into
    inbox_quick_reply_configs (
        inbox_id,
        welcome_message,
        transfer_keyword,
        queue_reply,
        assigned_reply,
        closed_reply,
        no_reply_timeout_reply,
        enabled
    )
values
    ($1, $2, $3, $4, $5, $6, $7, $8)
on conflict (inbox_id) do
update
set
    welcome_message = excluded.welcome_message,
    transfer_keyword = excluded.transfer_keyword,
    queue_reply = excluded.queue_reply,
    assigned_reply = excluded.assigned_reply,
    closed_reply = excluded.closed_reply,
    no_reply_timeout_reply = excluded.no_reply_timeout_reply,
    enabled = excluded.enabled,
    updated_at = now()
returning
    *;

-- name: delete-config
delete from
    inbox_quick_reply_configs
where
    inbox_id = $1;

-- name: get-topics
select
    id,
    inbox_id,
    name,
    names,
    hint_message,
    sort_order
from
    quick_reply_topics
where
    inbox_id = $1
order by
    sort_order asc,
    id asc;

-- name: get-topic
select
    id,
    inbox_id,
    name,
    names,
    hint_message,
    sort_order
from
    quick_reply_topics
where
    id = $1;

-- name: get-topic-by-name
select
    id,
    inbox_id,
    name,
    names,
    hint_message,
    sort_order
from
    quick_reply_topics
where
    inbox_id = $1
    and names && ARRAY[$2::text]
limit 1;

-- name: insert-topic
insert into
    quick_reply_topics (inbox_id, name, names, hint_message, sort_order)
values
    ($1, $2, $3, $4, $5)
returning
    *;

-- name: update-topic
update
    quick_reply_topics
set
    name = $2,
    names = $3,
    hint_message = $4,
    sort_order = $5,
    updated_at = now()
where
    id = $1
returning
    *;

-- name: delete-topic
delete from
    quick_reply_topics
where
    id = $1;

-- name: get-questions-by-topic
select
    id,
    topic_id,
    question,
    answer,
    sort_order
from
    quick_reply_questions
where
    topic_id = $1
order by
    sort_order asc,
    id asc;

-- name: get-question
select
    id,
    topic_id,
    question,
    answer,
    sort_order
from
    quick_reply_questions
where
    id = $1;

-- name: insert-question
insert into
    quick_reply_questions (topic_id, question, answer, sort_order)
values
    ($1, $2, $3, $4)
returning
    *;

-- name: update-question
update
    quick_reply_questions
set
    question = $2,
    answer = $3,
    sort_order = $4,
    updated_at = now()
where
    id = $1
returning
    *;

-- name: delete-question
delete from
    quick_reply_questions
where
    id = $1;
