-- name: GetSettings :one
SELECT id, access_log_retention_days, reports_dashboard, created_at, updated_at
FROM settings
ORDER BY created_at ASC
LIMIT 1;

-- name: InsertSettings :one
INSERT INTO settings (id, access_log_retention_days, reports_dashboard)
VALUES ($1, $2, $3)
RETURNING id, access_log_retention_days, reports_dashboard, created_at, updated_at;

-- name: UpdateSettings :one
UPDATE settings
SET
    access_log_retention_days = $2,
    updated_at = now()
WHERE id = $1
RETURNING id, access_log_retention_days, reports_dashboard, created_at, updated_at;

-- name: UpdateSettingsReportsDashboard :one
UPDATE settings
SET
    reports_dashboard = $2,
    updated_at = now()
WHERE id = $1
RETURNING id, access_log_retention_days, reports_dashboard, created_at, updated_at;
