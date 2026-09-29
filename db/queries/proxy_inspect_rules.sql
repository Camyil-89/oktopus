-- name: ListProxyInspectRules :many
SELECT
    id,
    name,
    script,
    action,
    enabled,
    sort_order,
    created_at,
    updated_at
FROM proxy_inspect_rules
ORDER BY sort_order ASC, created_at ASC;

-- name: ListProxyInspectRulesSummary :many
SELECT
    id,
    name,
    action,
    enabled,
    sort_order,
    length(script) AS script_length,
    created_at,
    updated_at
FROM proxy_inspect_rules
ORDER BY sort_order ASC, created_at ASC;

-- name: GetProxyInspectRule :one
SELECT
    id,
    name,
    script,
    action,
    enabled,
    sort_order,
    created_at,
    updated_at
FROM proxy_inspect_rules
WHERE id = $1;

-- name: CountProxyInspectRules :one
SELECT count(*)::bigint AS count FROM proxy_inspect_rules;

-- name: DeleteAllProxyInspectRules :exec
DELETE FROM proxy_inspect_rules;

-- name: DeleteProxyInspectRulesExcept :exec
DELETE FROM proxy_inspect_rules
WHERE cardinality(@keep_ids::uuid[]) = 0
   OR NOT (id = ANY (@keep_ids::uuid[]));

-- name: UpsertProxyInspectRule :one
INSERT INTO proxy_inspect_rules (
    id,
    name,
    script,
    action,
    enabled,
    sort_order
) VALUES (
    $1, $2, $3, $4, $5, $6
)
ON CONFLICT (id) DO UPDATE
SET
    name = EXCLUDED.name,
    script = EXCLUDED.script,
    action = EXCLUDED.action,
    enabled = EXCLUDED.enabled,
    sort_order = EXCLUDED.sort_order,
    updated_at = now()
RETURNING
    id,
    name,
    script,
    action,
    enabled,
    sort_order,
    created_at,
    updated_at;

-- name: InsertProxyInspectRule :one
INSERT INTO proxy_inspect_rules (
    id,
    name,
    script,
    action,
    enabled,
    sort_order
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING
    id,
    name,
    script,
    action,
    enabled,
    sort_order,
    created_at,
    updated_at;
