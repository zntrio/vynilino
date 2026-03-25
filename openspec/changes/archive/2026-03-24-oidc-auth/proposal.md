## Why

vynilino currently only supports local email/password authentication. Adding OIDC (OpenID Connect) support lets users authenticate via an external Identity Provider (e.g., Google, GitHub, self-hosted Keycloak/Dex), enabling SSO, federation, and delegated identity management without vynilino storing credentials. This is particularly valuable for self-hosters who already operate an IdP or who want to centralise access control across their home-lab services.

## What Changes

- **New OIDC provider adapter**: Discovery-based OIDC client (Authorization Code Flow with PKCE) that exchanges an authorization code for an ID token and extracts the subject claim as the user identity.
- **New GraphQL mutations**: `oidcAuthorizationURL` (returns the redirect URL + state/nonce) and `oidcCallback` (exchanges code → ID token → vynilino PASETO access + refresh tokens).
- **Account linking**: An OIDC-authenticated user is matched by `sub` claim (stored per-user); first login auto-provisions the account. Existing local accounts can optionally link their OIDC identity.
- **Configuration**: New env vars (`VYNILINO_OIDC_ISSUER`, `VYNILINO_OIDC_CLIENT_ID`, `VYNILINO_OIDC_CLIENT_SECRET`, `VYNILINO_OIDC_REDIRECT_URL`) — OIDC is disabled when `VYNILINO_OIDC_ISSUER` is unset.
- **Database migration**: New `oidc_identities` table (`user_id`, `provider`, `subject`, `created_at`).
- **Modified user-auth capability**: Adds OIDC login flow alongside existing email/password flow; local registration/login remain fully functional and are not removed.

## Capabilities

### New Capabilities

- `oidc-auth`: OIDC Authorization Code + PKCE flow, provider discovery, ID-token validation, account auto-provisioning, account linking, and vynilino token issuance for OIDC-authenticated users.

### Modified Capabilities

- `user-auth`: Extends the authentication surface to include the OIDC flow as an alternative login path. Adds new mutations to the GraphQL API contract. Existing email/password requirements are unchanged.

## Impact

- `internal/app/auth.go` — new `OIDCService` (or methods on `UserService`) for OIDC flows
- `internal/domain/user.go` — new `OIDCIdentity` entity and `OIDCIdentityRepository` interface
- `internal/adapter/storage/sqlite/` — new migration, new `oidc_identity_repo.go`, new sqlc queries
- `internal/adapter/graphql/graph/schema.graphql` — two new mutations, new input/output types
- `internal/config/config.go` — four new OIDC env vars
- `go.mod` — new dependency: `golang.org/x/oauth2` (OIDC client via `coreos/go-oidc/v3` or `golang.org/x/oauth2/oidc`)
- `README.md` — new OIDC configuration section
