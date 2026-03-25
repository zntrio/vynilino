-- name: CreateOIDCState :exec
INSERT INTO oidc_states (state, nonce, code_verifier, created_at)
VALUES (?, ?, ?, ?);

-- name: FindOIDCStateByState :one
SELECT * FROM oidc_states WHERE state = ? LIMIT 1;

-- name: DeleteOIDCState :exec
DELETE FROM oidc_states WHERE state = ?;

-- name: DeleteExpiredOIDCStates :exec
DELETE FROM oidc_states WHERE created_at < ?;
