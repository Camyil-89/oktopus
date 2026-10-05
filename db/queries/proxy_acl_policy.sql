-- name: GetProxyACLPolicyByInstance :one
SELECT id, instance_id, config_text, updated_at
FROM proxy_acl_policy
WHERE instance_id = $1;

-- name: UpsertProxyACLPolicyByInstance :one
UPDATE proxy_acl_policy
SET config_text = $2, updated_at = now()
WHERE instance_id = $1
RETURNING id, instance_id, config_text, updated_at;

-- name: ListProxyACLPolicyInstanceIDs :many
SELECT instance_id FROM proxy_acl_policy;
