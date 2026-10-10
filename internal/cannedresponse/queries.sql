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

-- name: get-user-personal-responses
select
    id,
    user_id,
    content,
    sort_order,
    created_at,
    updated_at
from
    user_canned_responses
where
    user_id = $1
order by
    sort_order asc,
    id asc;

-- name: get-user-personal-response
select
    id,
    user_id,
    content,
    sort_order,
    created_at,
    updated_at
from
    user_canned_responses
where
    id = $1
    and user_id = $2;

-- name: next-user-personal-sort-order
select
    coalesce(max(sort_order) + 1, 0)
from
    user_canned_responses
where
    user_id = $1;

-- name: insert-user-personal-response
insert into
    user_canned_responses (user_id, content, sort_order)
values
    ($1, $2, $3)
returning
    *;

-- name: update-user-personal-response
update
    user_canned_responses
set
    content = $2,
    sort_order = $3,
    updated_at = now()
where
    id = $1
    and user_id = $4
returning
    *;

-- name: delete-user-personal-response
delete from
    user_canned_responses
where
    id = $1
    and user_id = $2;

-- name: get-user-canned-pins
select
    source,
    response_id,
    sort_order
from
    user_canned_pins
where
    user_id = $1
order by
    sort_order asc,
    created_at asc;

-- name: insert-user-canned-pin
insert into
    user_canned_pins (user_id, source, response_id, sort_order)
values
    ($1, $2, $3, $4)
on conflict (user_id, source, response_id) do nothing;

-- name: update-user-canned-pin-sort
update
    user_canned_pins
set
    sort_order = $2
where
    user_id = $1
    and source = $3
    and response_id = $4;

-- name: delete-user-canned-pin
delete from
    user_canned_pins
where
    user_id = $1
    and source = $2
    and response_id = $3;

-- name: next-user-canned-pin-sort-order
select
    coalesce(max(sort_order) + 1, 0)
from
    user_canned_pins
where
    user_id = $1;
