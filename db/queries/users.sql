-- name: GetUserByUsername :one
SELECT id, username, password_hash, enabled, created_at, updated_at
FROM users
WHERE username = $1;

-- name: GetUserByID :one
SELECT id, username, password_hash, enabled, created_at, updated_at
FROM users
WHERE id = $1;

-- name: CountUsers :one
SELECT count(*)::bigint
FROM users
WHERE (@search::text = '' OR username ILIKE '%' || @search || '%');

-- name: ListUsers :many
SELECT id, username, password_hash, enabled, created_at, updated_at
FROM users
WHERE (@search::text = '' OR username ILIKE '%' || @search || '%')
ORDER BY username ASC
LIMIT @limit_val OFFSET @offset_val;

-- name: CreateUser :one
INSERT INTO users (id, username, password_hash, enabled)
VALUES ($1, $2, $3, $4)
RETURNING id, username, password_hash, enabled, created_at, updated_at;

-- name: UpdateUserEnabled :exec
UPDATE users
SET enabled = $2, updated_at = now()
WHERE id = $1;

-- name: UpdateUserPassword :exec
UPDATE users
SET password_hash = $2, updated_at = now()
WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;
