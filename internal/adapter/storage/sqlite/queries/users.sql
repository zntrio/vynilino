-- name: CreateUser :one
INSERT INTO users (id, email, password_hash, role, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = ? LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ? LIMIT 1;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: UpdateLoginFailure :exec
UPDATE users
SET failed_login_count = failed_login_count + 1,
    locked_until = ?,
    updated_at = ?
WHERE id = ?;

-- name: ResetLoginFailure :exec
UPDATE users
SET failed_login_count = 0,
    locked_until = NULL,
    updated_at = ?
WHERE id = ?;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at ASC;

-- name: DeactivateUser :exec
UPDATE users SET active = 0, updated_at = ? WHERE email = ?;

-- name: ActivateUser :exec
UPDATE users SET active = 1, updated_at = ? WHERE email = ?;

-- name: ChangeUserPassword :exec
UPDATE users SET password_hash = ?, updated_at = ? WHERE email = ?;
