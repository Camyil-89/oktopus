-- +goose Up
ALTER TABLE settings
    ADD COLUMN reports_dashboard JSONB NOT NULL DEFAULT '{"version":1,"tabs":[]}'::jsonb;

-- +goose Down
ALTER TABLE settings DROP COLUMN IF EXISTS reports_dashboard;
