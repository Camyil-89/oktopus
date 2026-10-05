-- name: ListProxyInspectRuleInstanceIDs :many
SELECT DISTINCT instance_id FROM proxy_inspect_rules;

-- name: ListProxyInspectRulesByInstance :many
SELECT
    id, instance_id, name, script, action, enabled, sort_order, created_at, updated_at
FROM proxy_inspect_rules
WHERE instance_id = $1
ORDER BY sort_order ASC, created_at ASC;

-- name: ListProxyInspectRulesSummaryByInstance :many
SELECT
    id, instance_id, name, action, enabled, sort_order,
    length(script) AS script_length, created_at, updated_at
FROM proxy_inspect_rules
WHERE instance_id = $1
ORDER BY sort_order ASC, created_at ASC;

-- name: GetProxyInspectRule :one
SELECT
    id, instance_id, name, script, action, enabled, sort_order, created_at, updated_at
FROM proxy_inspect_rules
WHERE id = $1;

-- name: CountProxyInspectRulesByInstance :one
SELECT count(*)::bigint AS count FROM proxy_inspect_rules WHERE instance_id = $1;

-- name: DeleteProxyInspectRulesByInstanceExcept :exec
DELETE FROM proxy_inspect_rules
WHERE instance_id = $1
  AND (cardinality(@keep_ids::uuid[]) = 0 OR NOT (id = ANY (@keep_ids::uuid[])));

-- name: UpsertProxyInspectRule :one
INSERT INTO proxy_inspect_rules (
    id, instance_id, name, script, action, enabled, sort_order
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
ON CONFLICT (id) DO UPDATE
SET
    instance_id = EXCLUDED.instance_id,
    name = EXCLUDED.name,
    script = EXCLUDED.script,
    action = EXCLUDED.action,
    enabled = EXCLUDED.enabled,
    sort_order = EXCLUDED.sort_order,
    updated_at = now()
RETURNING
    id, instance_id, name, script, action, enabled, sort_order, created_at, updated_at;
