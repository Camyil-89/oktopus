-- name: GetProxySettings :one
SELECT
    id,
    proxy_enabled,
    listen,
    connect_mode,
    ca_cert_path,
    ca_key_path,
    auth_enabled,
    auth_static_users,
    auth_realm,
    auth_backend,
    auth_cache_ttl_minutes,
    ldap_url,
    ldap_base_dn,
    ldap_bind_dn,
    ldap_bind_password,
    access_log_retention_days,
    created_at,
    updated_at
FROM proxy_settings
ORDER BY created_at ASC
LIMIT 1;

-- name: InsertProxySettings :one
INSERT INTO proxy_settings (
    id,
    proxy_enabled,
    listen,
    connect_mode,
    ca_cert_path,
    ca_key_path,
    auth_enabled,
    auth_static_users,
    auth_realm,
    auth_backend,
    auth_cache_ttl_minutes,
    ldap_url,
    ldap_base_dn,
    ldap_bind_dn,
    ldap_bind_password,
    access_log_retention_days
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,
    $12, $13, $14, $15, $16
)
RETURNING
    id,
    proxy_enabled,
    listen,
    connect_mode,
    ca_cert_path,
    ca_key_path,
    auth_enabled,
    auth_static_users,
    auth_realm,
    auth_backend,
    auth_cache_ttl_minutes,
    ldap_url,
    ldap_base_dn,
    ldap_bind_dn,
    ldap_bind_password,
    access_log_retention_days,
    created_at,
    updated_at;

-- name: UpdateProxySettings :one
UPDATE proxy_settings
SET
    proxy_enabled = $2,
    listen = $3,
    connect_mode = $4,
    ca_cert_path = $5,
    ca_key_path = $6,
    auth_enabled = $7,
    auth_static_users = $8,
    auth_realm = $9,
    auth_backend = $10,
    auth_cache_ttl_minutes = $11,
    ldap_url = $12,
    ldap_base_dn = $13,
    ldap_bind_dn = $14,
    ldap_bind_password = $15,
    access_log_retention_days = $16,
    updated_at = now()
WHERE id = $1
RETURNING
    id,
    proxy_enabled,
    listen,
    connect_mode,
    ca_cert_path,
    ca_key_path,
    auth_enabled,
    auth_static_users,
    auth_realm,
    auth_backend,
    auth_cache_ttl_minutes,
    ldap_url,
    ldap_base_dn,
    ldap_bind_dn,
    ldap_bind_password,
    access_log_retention_days,
    created_at,
    updated_at;

-- name: UpdateProxyCAPaths :one
UPDATE proxy_settings
SET
    ca_cert_path = $2,
    ca_key_path = $3,
    updated_at = now()
WHERE id = $1
RETURNING
    id,
    proxy_enabled,
    listen,
    connect_mode,
    ca_cert_path,
    ca_key_path,
    auth_enabled,
    auth_static_users,
    auth_realm,
    auth_backend,
    auth_cache_ttl_minutes,
    ldap_url,
    ldap_base_dn,
    ldap_bind_dn,
    ldap_bind_password,
    access_log_retention_days,
    created_at,
    updated_at;
