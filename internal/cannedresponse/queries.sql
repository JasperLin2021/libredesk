-- name: get-all-categories
select
    id,
    parent_id,
    name,
    sort_order,
    created_at,
    updated_at
from
    canned_categories
order by
    sort_order asc,
    id asc;

-- name: get-category
select
    id,
    parent_id,
    name,
    sort_order,
    created_at,
    updated_at
from
    canned_categories
where
    id = $1;

-- name: insert-category
insert into
    canned_categories (parent_id, name, sort_order)
values
    ($1, $2, $3)
returning
    *;

-- name: update-category
update
    canned_categories
set
    parent_id = $2,
    name = $3,
    sort_order = $4,
    updated_at = now()
where
    id = $1
returning
    *;

-- name: delete-category
delete from
    canned_categories
where
    id = $1;

-- name: next-category-sort-order
select
    coalesce(max(sort_order) + 1, 0)
from
    canned_categories
where
    parent_id is not distinct from $1;

-- name: get-all-responses
select
    id,
    category_id,
    content,
    sort_order,
    created_at,
    updated_at
from
    canned_responses
order by
    sort_order asc,
    id asc;

-- name: get-response
select
    id,
    category_id,
    content,
    sort_order,
    created_at,
    updated_at
from
    canned_responses
where
    id = $1;

-- name: insert-response
insert into
    canned_responses (category_id, content, sort_order)
values
    ($1, $2, $3)
returning
    *;

-- name: update-response
update
    canned_responses
set
    content = $2,
    sort_order = $3,
    updated_at = now()
where
    id = $1
returning
    *;

-- name: delete-response
delete from
    canned_responses
where
    id = $1;

-- name: next-response-sort-order
select
    coalesce(max(sort_order) + 1, 0)
from
    canned_responses
where
    category_id = $1;

-- name: delete-all-responses
delete from
    canned_responses;

-- name: delete-all-categories
delete from
    canned_categories;
