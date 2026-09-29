CREATE TABLE proxy_settings (
    id UUID PRIMARY KEY,
    proxy_enabled BOOLEAN NOT NULL DEFAULT false,
    listen TEXT NOT NULL,
    connect_mode TEXT NOT NULL,
    ca_cert_path TEXT NOT NULL,
    ca_key_path TEXT NOT NULL,
    auth_enabled BOOLEAN NOT NULL,
    auth_static_users TEXT NOT NULL,
    auth_realm TEXT NOT NULL,
    auth_backend TEXT NOT NULL,
    auth_cache_ttl_minutes BIGINT NOT NULL,
    ldap_url TEXT NOT NULL,
    ldap_base_dn TEXT NOT NULL,
    ldap_bind_dn TEXT NOT NULL,
    ldap_bind_password TEXT NOT NULL,
    access_log_retention_days INTEGER NOT NULL DEFAULT 3
        CHECK (access_log_retention_days = 0 OR access_log_retention_days >= 1),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
