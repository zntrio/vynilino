-- name: CreateOIDCIdentity :exec
INSERT INTO oidc_identities (user_id, provider, subject, created_at)
VALUES (?, ?, ?, ?);

-- name: FindOIDCIdentityByProviderSubject :one
SELECT * FROM oidc_identities WHERE provider = ? AND subject = ? LIMIT 1;
