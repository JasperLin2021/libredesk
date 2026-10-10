-- name: get-all-categories
select
    c.id,
    c.parent_id,
    c.code,
    c.name,
    c.sort_order,
    c.created_at,
    c.updated_at,
    coalesce((
        select count(*) from kb_articles a where a.category_id = c.id
    ), 0)::int as article_count
from
    kb_categories c
order by
    c.sort_order asc,
    c.id asc;

-- name: get-category
select
    id,
    parent_id,
    code,
    name,
    sort_order,
    created_at,
    updated_at
from
    kb_categories
where
    id = $1;

-- name: get-category-by-code
select
    id,
    parent_id,
    code,
    name,
    sort_order,
    created_at,
    updated_at
from
    kb_categories
where
    parent_id is not distinct from $1
    and code = $2;

-- name: insert-category
insert into
    kb_categories (parent_id, code, name, sort_order)
values
    ($1, $2, $3, $4)
returning
    *;

-- name: get-category-by-parent-and-name
-- Lookup used by the importer to recover the row id when
-- insert-category-on-conflict returns no rows (i.e. a conflict on the
-- `(parent_id, name)` unique index already happened).
select
    id,
    parent_id,
    code,
    name,
    sort_order,
    created_at,
    updated_at
from
    kb_categories
where
    parent_id is not distinct from $1
    and name = $2;

-- name: insert-category-on-conflict
-- Idempotent insert used by the bulk importer. Uses the
-- `(COALESCE(parent_id, 0), name)` expression unique index to silently
-- skip rows that already exist. When 0 rows are returned the caller
-- falls back to a lookup query to recover the existing id without
-- poisoning the surrounding transaction.
insert into
    kb_categories (parent_id, code, name, sort_order)
values
    ($1, $2, $3, $4)
on conflict (coalesce(parent_id, 0), name) do nothing
returning
    id;

-- name: update-category
update
    kb_categories
set
    parent_id = $2,
    code = $3,
    name = $4,
    sort_order = $5,
    updated_at = now()
where
    id = $1
returning
    *;

-- name: delete-category
delete from
    kb_categories
where
    id = $1;

-- name: next-category-sort-order
select
    coalesce(max(sort_order) + 1, 0)
from
    kb_categories
where
    parent_id is not distinct from $1;

-- name: count-articles-by-category
select
    coalesce(count(*) filter (where category_id is not null), 0)::int as direct_count
from
    kb_articles
where
    category_id = $1
    or category_id in (
        select id from kb_categories where parent_id = $1
    );

-- name: article-exists-in-other-categories
-- Detects whether an article (by external_id) is already attached to a
-- different category, used by the bulk importer to refuse multi-category
-- associations for the same row.
select exists (
    select 1 from kb_articles where external_id = $1 and (id <> $2 or category_id is distinct from $3)
) as conflict;

-- name: get-article-by-external-id
select
    id,
    category_id,
    external_id,
    code,
    title,
    author,
    content,
    status,
    sort_order,
    created_at,
    updated_at
from
    kb_articles
where
    external_id = $1;

-- name: get-article
select
    a.id,
    a.category_id,
    a.external_id,
    a.code,
    a.title,
    a.author,
    a.content,
    a.status,
    a.sort_order,
    a.created_at,
    a.updated_at,
    pc.id as parent_category_id,
    pc.name as parent_category_name,
    c.name as category_name
from
    kb_articles a
    left join kb_categories c on c.id = a.category_id
    left join kb_categories pc on pc.id = c.parent_id
where
    a.id = $1;

-- name: get-article-attachments
select
    id,
    article_id,
    media_uuid,
    file_name,
    content_type,
    size,
    created_at
from
    kb_article_attachments
where
    article_id = $1
order by
    id asc;

-- name: list-articles
-- Paginated knowledge article listing with category, date, status and
-- keyword filters. The keyword scope is selected by the caller; the SQL
-- is parameterised through `keywords` (an array of search terms) and
-- `keyword_field` ('all'|'title'|'content').
--
-- `parent_category_id` is a virtual filter: when set, it scopes the list
-- to all articles under any child of the given level-1 category.
select
    a.id,
    a.category_id,
    a.external_id,
    a.code,
    a.title,
    a.author,
    a.status,
    a.sort_order,
    a.created_at,
    a.updated_at,
    c.name as category_name,
    pc.id as parent_category_id,
    pc.name as parent_category_name,
    -- Strip HTML tags and truncate for the list summary.
    case
        when a.content is null then ''
        when length(regexp_replace(a.content, '<[^>]*>', '', 'g')) <= 200
            then regexp_replace(a.content, '<[^>]*>', '', 'g')
        else substr(regexp_replace(a.content, '<[^>]*>', '', 'g'), 1, 200) || '…'
    end as summary
from
    kb_articles a
    left join kb_categories c on c.id = a.category_id
    left join kb_categories pc on pc.id = c.parent_id
where
    ($1::bigint is null or a.category_id = $1)
    and ($2::bigint is null or pc.id = $2::bigint or a.category_id = $2::bigint)
    and ($3::text is null or a.status = $3)
    and ($4::timestamptz is null or (case when $12::text = 'updated_at' then a.updated_at else a.created_at end) >= $4)
    and ($5::timestamptz is null or (case when $12::text = 'updated_at' then a.updated_at else a.created_at end) < $5)
    and (
        $6::text is null
        or (
            $7::text = 'title'
            and exists (
                select 1 from unnest($8::text[]) kw where a.title ilike '%' || kw || '%'
            )
        )
        or (
            $7::text = 'content'
            and exists (
                select 1 from unnest($8::text[]) kw where a.content ilike '%' || kw || '%'
            )
        )
        or (
            $7::text = 'all'
            and exists (
                select 1 from unnest($8::text[]) kw
                where a.title ilike '%' || kw || '%' or a.content ilike '%' || kw || '%'
            )
        )
    )
order by
    case when $9::text = 'asc' then a.created_at end asc,
    case when $9::text = 'desc' then a.created_at end desc,
    a.id desc
limit $10 offset $11;

-- name: count-articles
-- Returns the total number of articles matching the same filter as
-- list-articles. Kept separate so the manager can return both the page
-- and the total in two round-trips.
select
    count(*)::int as total
from
    kb_articles a
    left join kb_categories c on c.id = a.category_id
    left join kb_categories pc on pc.id = c.parent_id
where
    ($1::bigint is null or a.category_id = $1)
    and ($2::bigint is null or pc.id = $2::bigint or a.category_id = $2::bigint)
    and ($3::text is null or a.status = $3)
    and ($4::timestamptz is null or (case when $9::text = 'updated_at' then a.updated_at else a.created_at end) >= $4)
    and ($5::timestamptz is null or (case when $9::text = 'updated_at' then a.updated_at else a.created_at end) < $5)
    and (
        $6::text is null
        or (
            $7::text = 'title'
            and exists (
                select 1 from unnest($8::text[]) kw where a.title ilike '%' || kw || '%'
            )
        )
        or (
            $7::text = 'content'
            and exists (
                select 1 from unnest($8::text[]) kw where a.content ilike '%' || kw || '%'
            )
        )
        or (
            $7::text = 'all'
            and exists (
                select 1 from unnest($8::text[]) kw
                where a.title ilike '%' || kw || '%' or a.content ilike '%' || kw || '%'
            )
        )
    );

-- name: insert-article
insert into
    kb_articles (
        category_id,
        external_id,
        code,
        title,
        author,
        content,
        status,
        sort_order
    )
values
    ($1, $2, $3, $4, $5, $6, $7, $8)
returning
    *;

-- name: insert-article-on-conflict
-- Idempotent insert used by the bulk importer. Uses the partial unique
-- index on `external_id` to skip rows that already exist, avoiding a
-- unique-violation that would otherwise abort the transaction.
insert into
    kb_articles (
        category_id,
        external_id,
        code,
        title,
        author,
        content,
        status,
        sort_order
    )
values
    ($1, $2, $3, $4, $5, $6, $7, $8)
on conflict (external_id) where external_id is not null do nothing
returning
    id;

-- name: update-article
update
    kb_articles
set
    category_id = $2,
    external_id = $3,
    code = $4,
    title = $5,
    author = $6,
    content = $7,
    status = $8,
    sort_order = $9,
    updated_at = now()
where
    id = $1
returning
    *;

-- name: delete-article
delete from
    kb_articles
where
    id = $1;

-- name: insert-article-attachment
insert into
    kb_article_attachments (article_id, media_uuid, file_name, content_type, size)
values
    ($1, $2, $3, $4, $5);

-- name: delete-article-attachments
delete from
    kb_article_attachments
where
    article_id = $1;

-- name: list-all-articles
-- Used by the bulk exporter to stream every article. No pagination so
-- the full dataset is fetched in a single query.
select
    a.id,
    a.external_id,
    a.code,
    a.title,
    a.author,
    a.status,
    a.created_at,
    a.updated_at,
    coalesce(pc.name, '') as parent_category_name,
    coalesce(c.name, '') as category_name,
    coalesce(a.content, '') as content
from
    kb_articles a
    left join kb_categories c on c.id = a.category_id
    left join kb_categories pc on pc.id = c.parent_id
order by
    a.id asc;
