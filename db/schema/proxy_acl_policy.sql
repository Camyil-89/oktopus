CREATE TABLE proxy_acl_policy (
    id UUID PRIMARY KEY,
    instance_id UUID NOT NULL REFERENCES proxy_instances (id) ON DELETE CASCADE,
    config_text TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT proxy_acl_policy_instance_unique UNIQUE (instance_id)
);
