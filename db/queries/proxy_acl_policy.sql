-- name: GetProxyACLPolicy :one
SELECT id, config_text, updated_at
FROM proxy_acl_policy
ORDER BY id
LIMIT 1;

-- name: UpsertProxyACLPolicy :one
UPDATE proxy_acl_policy
SET config_text = $1, updated_at = now()
WHERE id = (SELECT id FROM proxy_acl_policy ORDER BY id LIMIT 1)
RETURNING id, config_text, updated_at;
