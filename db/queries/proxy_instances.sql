-- name: ListProxyInstances :many
SELECT
    id, name, enabled, listen, connect_mode, ca_cert_path, ca_key_path,
    auth_enabled, auth_static_users, auth_realm, auth_backend, auth_cache_ttl_minutes,
    ldap_url, ldap_base_dn, ldap_bind_dn, ldap_bind_password, sort_order,
    created_at, updated_at
FROM proxy_instances
ORDER BY sort_order ASC, created_at ASC;

-- name: GetProxyInstance :one
SELECT
    id, name, enabled, listen, connect_mode, ca_cert_path, ca_key_path,
    auth_enabled, auth_static_users, auth_realm, auth_backend, auth_cache_ttl_minutes,
    ldap_url, ldap_base_dn, ldap_bind_dn, ldap_bind_password, sort_order,
    created_at, updated_at
FROM proxy_instances
WHERE id = $1;

-- name: InsertProxyInstance :one
INSERT INTO proxy_instances (
    id, name, enabled, listen, connect_mode, ca_cert_path, ca_key_path,
    auth_enabled, auth_static_users, auth_realm, auth_backend, auth_cache_ttl_minutes,
    ldap_url, ldap_base_dn, ldap_bind_dn, ldap_bind_password, sort_order
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
)
RETURNING
    id, name, enabled, listen, connect_mode, ca_cert_path, ca_key_path,
    auth_enabled, auth_static_users, auth_realm, auth_backend, auth_cache_ttl_minutes,
    ldap_url, ldap_base_dn, ldap_bind_dn, ldap_bind_password, sort_order,
    created_at, updated_at;

-- name: UpdateProxyInstance :one
UPDATE proxy_instances
SET
    name = $2,
    enabled = $3,
    listen = $4,
    connect_mode = $5,
    ca_cert_path = $6,
    ca_key_path = $7,
    auth_enabled = $8,
    auth_static_users = $9,
    auth_realm = $10,
    auth_backend = $11,
    auth_cache_ttl_minutes = $12,
    ldap_url = $13,
    ldap_base_dn = $14,
    ldap_bind_dn = $15,
    ldap_bind_password = $16,
    sort_order = $17,
    updated_at = now()
WHERE id = $1
RETURNING
    id, name, enabled, listen, connect_mode, ca_cert_path, ca_key_path,
    auth_enabled, auth_static_users, auth_realm, auth_backend, auth_cache_ttl_minutes,
    ldap_url, ldap_base_dn, ldap_bind_dn, ldap_bind_password, sort_order,
    created_at, updated_at;

-- name: UpdateProxyInstanceCAPaths :one
UPDATE proxy_instances
SET ca_cert_path = $2, ca_key_path = $3, updated_at = now()
WHERE id = $1
RETURNING
    id, name, enabled, listen, connect_mode, ca_cert_path, ca_key_path,
    auth_enabled, auth_static_users, auth_realm, auth_backend, auth_cache_ttl_minutes,
    ldap_url, ldap_base_dn, ldap_bind_dn, ldap_bind_password, sort_order,
    created_at, updated_at;

-- name: DeleteProxyInstance :exec
DELETE FROM proxy_instances WHERE id = $1;

-- name: InsertProxyACLPolicyForInstance :one
INSERT INTO proxy_acl_policy (id, instance_id, config_text)
VALUES ($1, $2, '')
RETURNING id, instance_id, config_text, updated_at;
