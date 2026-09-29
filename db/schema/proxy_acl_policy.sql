CREATE TABLE proxy_acl_policy (
    id UUID PRIMARY KEY,
    config_text TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
