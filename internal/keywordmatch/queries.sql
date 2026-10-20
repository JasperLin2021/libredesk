-- name: get-all-rules
select
    *
from
    keyword_match_rules
order by
    sort_order asc,
    id asc;

-- name: get-enabled-rules
select
    *
from
    keyword_match_rules
where
    enabled = true
order by
    sort_order asc,
    id asc;

-- name: get-rule
select
    *
from
    keyword_match_rules
where
    id = $1;

-- name: insert-rule
insert into
    keyword_match_rules (
        name,
        source,
        match_type,
        pattern,
        case_sensitive,
        search_first,
        action,
        reply_content,
        enabled,
        sort_order
    )
values
    ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
returning
    *;

-- name: update-rule
update
    keyword_match_rules
set
    name = $2,
    source = $3,
    match_type = $4,
    pattern = $5,
    case_sensitive = $6,
    search_first = $7,
    action = $8,
    reply_content = $9,
    enabled = $10,
    sort_order = $11,
    updated_at = now()
where
    id = $1
returning
    *;

-- name: delete-rule
delete from
    keyword_match_rules
where
    id = $1;
