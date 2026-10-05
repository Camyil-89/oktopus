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
