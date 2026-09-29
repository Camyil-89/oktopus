-- name: ListProxyACLLists :many
SELECT id, name, list_type, body, source_mode, source_url, poll_interval_minutes, created_at, updated_at
FROM proxy_acl_lists
ORDER BY name, id;

-- name: ListProxyACLListsSummary :many
SELECT
    id,
    name,
    list_type,
    source_mode,
    source_url,
    poll_interval_minutes,
    created_at,
    updated_at,
    CASE
        WHEN body = '' THEN 0
        ELSE length(body) - length(replace(body, E'\n', '')) + 1
    END::integer AS body_line_count,
    left(
        btrim(replace(split_part(body || E'\n', E'\n', 1), E'\r', ''), E' \t'),
        48
    ) AS body_preview
FROM proxy_acl_lists
ORDER BY name, id;

-- name: GetProxyACLList :one
SELECT id, name, list_type, body, source_mode, source_url, poll_interval_minutes, created_at, updated_at
FROM proxy_acl_lists
WHERE id = $1;

-- name: ListProxyACLListsByNames :many
SELECT id, name, list_type, body, source_mode, source_url, poll_interval_minutes, created_at, updated_at
FROM proxy_acl_lists
WHERE cardinality(@names::text[]) > 0
  AND name = ANY (@names::text[])
ORDER BY name, id;

-- name: DeleteAllProxyACLLists :exec
DELETE FROM proxy_acl_lists;

-- name: DeleteProxyACLListsExcept :exec
DELETE FROM proxy_acl_lists
WHERE cardinality(@keep_ids::uuid[]) = 0
   OR NOT (id = ANY (@keep_ids::uuid[]));

-- name: UpsertProxyACLList :one
INSERT INTO proxy_acl_lists (id, name, list_type, body, source_mode, source_url, poll_interval_minutes)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (name) DO UPDATE
SET
    list_type = EXCLUDED.list_type,
    body = EXCLUDED.body,
    source_mode = EXCLUDED.source_mode,
    source_url = EXCLUDED.source_url,
    poll_interval_minutes = EXCLUDED.poll_interval_minutes,
    updated_at = now()
RETURNING id, name, list_type, body, source_mode, source_url, poll_interval_minutes, created_at, updated_at;

-- name: UpdateProxyACLListBody :one
UPDATE proxy_acl_lists
SET body = $2, updated_at = now()
WHERE id = $1
RETURNING id, name, list_type, body, source_mode, source_url, poll_interval_minutes, created_at, updated_at;
