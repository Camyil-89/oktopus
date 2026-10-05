CREATE TABLE proxy_instances (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT false,
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
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT proxy_instances_listen_unique UNIQUE (listen),
    CONSTRAINT proxy_instances_name_unique UNIQUE (name)
);

CREATE INDEX proxy_instances_sort_idx ON proxy_instances (sort_order, created_at);
