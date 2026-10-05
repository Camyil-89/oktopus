CREATE TABLE settings (
    id UUID PRIMARY KEY,
    access_log_retention_days INTEGER NOT NULL DEFAULT 3
        CHECK (access_log_retention_days = 0 OR access_log_retention_days >= 1),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
