-- +goose Up
CREATE TABLE users (
    id UUID PRIMARY KEY,
    username TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT users_username_unique UNIQUE (username)
);

CREATE INDEX users_username_idx ON users (username);

CREATE TABLE settings (
    id UUID PRIMARY KEY,
    access_log_retention_days INTEGER NOT NULL DEFAULT 3
        CHECK (access_log_retention_days = 0 OR access_log_retention_days >= 1),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

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

CREATE TABLE proxy_inspect_rules (
    id UUID PRIMARY KEY,
    instance_id UUID NOT NULL REFERENCES proxy_instances (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    script TEXT NOT NULL,
    action SMALLINT NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT true,
    sort_order INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT proxy_inspect_rules_action_check CHECK (action IN (0, 1))
);

CREATE INDEX proxy_inspect_rules_instance_sort_idx ON proxy_inspect_rules (instance_id, sort_order);

CREATE TABLE proxy_acl_policy (
    id UUID PRIMARY KEY,
    instance_id UUID NOT NULL REFERENCES proxy_instances (id) ON DELETE CASCADE,
    config_text TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT proxy_acl_policy_instance_unique UNIQUE (instance_id)
);

CREATE TABLE proxy_acl_lists (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    list_type TEXT NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    source_mode TEXT NOT NULL DEFAULT 'manual',
    source_url TEXT NOT NULL DEFAULT '',
    poll_interval_minutes INT NOT NULL DEFAULT 60,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT proxy_acl_lists_list_type_check CHECK (list_type IN ('src', 'dstdomain', 'port')),
    CONSTRAINT proxy_acl_lists_source_mode_check CHECK (source_mode IN ('manual', 'remote')),
    CONSTRAINT proxy_acl_lists_name_unique UNIQUE (name)
);

CREATE INDEX proxy_acl_lists_name_idx ON proxy_acl_lists (name);

-- +goose Down
DROP TABLE IF EXISTS proxy_acl_lists;
DROP TABLE IF EXISTS proxy_acl_policy;
DROP TABLE IF EXISTS proxy_inspect_rules;
DROP TABLE IF EXISTS proxy_instances;
DROP TABLE IF EXISTS settings;
DROP TABLE IF EXISTS users;
