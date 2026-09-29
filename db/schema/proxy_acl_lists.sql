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
