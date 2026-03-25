-- OIDC identity links: maps (provider, subject) → vynilino user_id
CREATE TABLE IF NOT EXISTS oidc_identities (
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider   TEXT NOT NULL,  -- issuer URL
    subject    TEXT NOT NULL,  -- sub claim
    created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    PRIMARY KEY (provider, subject)
);

-- Short-lived OIDC flow state for Authorization Code + PKCE
CREATE TABLE IF NOT EXISTS oidc_states (
    state          TEXT PRIMARY KEY NOT NULL,
    nonce          TEXT NOT NULL,
    code_verifier  TEXT NOT NULL,
    created_at     DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
