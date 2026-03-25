## Why

When OIDC is the only authentication method (no local passwords), showing a login form with an "OIDC Login" button creates unnecessary friction — the user has no other option. Operators need a way to force immediate redirect to the OAuth provider, bypassing the login form entirely.

## What Changes

- Add a new boolean config option `VYNILINO_OIDC_AUTO_REDIRECT` (default: `false`).
- When set to `true` and OIDC is enabled, `GET /login` SHALL respond with an HTTP 302 redirect directly to the OIDC authorization URL — the login page HTML is never sent to the client.
- If the OIDC authorization URL cannot be generated (provider unreachable, misconfiguration), the server SHALL return an HTTP 500 error page. There is no fallback to the login form; the operator must fix the OIDC provider or unset the flag.
- When `VYNILINO_OIDC_AUTO_REDIRECT` is `false` or OIDC is disabled, `GET /login` behaves exactly as today.

## Capabilities

### New Capabilities

- `oauth-auto-redirect`: Server-side redirect on `GET /login` to the OIDC provider when the flag is set, with a hard error if the provider is unavailable.

### Modified Capabilities

- `oidc-auth`: The OIDC provider configuration requirements expand to include the `VYNILINO_OIDC_AUTO_REDIRECT` flag and its server-side behavioral contract.

## Impact

- **Config**: `internal/config/config.go` — add `OIDCAutoRedirect bool` field.
- **Router**: `internal/adapter/graphql/server.go` — register an explicit `GET /login` chi route before the SPA mount; it intercepts the request and redirects when the flag is active.
- **Frontend**: no changes — `Login.js` and `login.html` are untouched.
- **No breaking changes**: existing deployments without the env var behave identically.
