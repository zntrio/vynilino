-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetRefreshTokenByHash :one
SELECT * FROM refresh_tokens WHERE token_hash = ? AND revoked = 0 LIMIT 1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens SET revoked = 1 WHERE id = ?;

-- name: RevokeAllUserTokens :exec
UPDATE refresh_tokens SET revoked = 1 WHERE user_id = ?;

-- name: DeleteExpiredTokens :exec
DELETE FROM refresh_tokens WHERE expires_at < ?;
