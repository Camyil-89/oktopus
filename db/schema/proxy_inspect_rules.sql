CREATE TABLE proxy_inspect_rules (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    script TEXT NOT NULL,
    action SMALLINT NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT true,
    sort_order INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT proxy_inspect_rules_action_check CHECK (action IN (0, 1))
);

CREATE INDEX proxy_inspect_rules_sort_idx ON proxy_inspect_rules (sort_order);
