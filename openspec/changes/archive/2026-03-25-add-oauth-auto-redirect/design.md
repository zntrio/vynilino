## Context

Vynilino's login page (`/login`) is a standalone HTML bundle served by `ui.Handler.SPAHandler()`. When an operator configures OIDC as the sole authentication method, users still see the local-credential form and must click an "OIDC Login" button before being redirected. The goal is to eliminate that step entirely via a server-side HTTP redirect.

`OIDCService.AuthorizationURL(ctx)` already exists and does everything needed: generates PKCE, stores state with TTL, returns the redirect URL. No new service logic is required.

## Goals / Non-Goals

**Goals:**
- Add `VYNILINO_OIDC_AUTO_REDIRECT` env flag (default: `false`).
- When active, `GET /login` returns HTTP 302 to the OIDC authorization URL without sending the login page HTML.
- When OIDC URL generation fails, return HTTP 500 — no silent fallback to the login form.

**Non-Goals:**
- Any frontend changes — `Login.js` and `login.html` are untouched.
- A new `/api/config` public endpoint — not needed.
- Disabling the `/graphql` `login` mutation (local auth remains accessible for direct API use).
- Supporting multiple OIDC providers.

## Decisions

### Decision 1: explicit chi route over modifying SPAHandler

**Chosen**: Register `r.Get("/login", loginRedirectHandler(cfg, oidcSvc, staticFiles))` in `server.go` before `r.Mount("/", uiHandler.SPAHandler())`. Chi gives explicit routes priority over a catch-all mount, so the intercept is clean.

**Alternatives considered**:
- *Modify `SPAHandler` to accept `OIDCService`*: mixes SPA serving concerns with OIDC logic; the handler becomes harder to test and reason about.
- *New middleware on all routes*: overly broad; only `GET /login` needs special handling.

### Decision 2: hard HTTP 500 on OIDC failure, no fallback

**Chosen**: If `AuthorizationURL` returns an error, the handler responds with HTTP 500 and a minimal HTML error page ("OIDC provider unavailable. Contact your administrator."). The login form is not served.

**Rationale**: `VYNILINO_OIDC_AUTO_REDIRECT=true` is an explicit operator decision that OIDC is the auth path. Silently falling back to the local form would undermine that intent and could expose a credential login surface the operator does not want. The only recovery path is an admin action: fix the OIDC provider or unset the flag.

Both transient errors (provider temporarily unreachable) and permanent errors (misconfiguration) result in the same 500 — the distinction is in the server logs, not the user-facing message.

**Alternatives considered**:
- *Serve login.html as fallback*: undermines operator intent; rejected.
- *Redirect to `/login?error=...`*: would loop back into the same handler unless it checks for the query param, adding complexity for no benefit.

### Decision 3: handler receives `*OIDCService` (nullable)

`NewOIDCService` already returns `nil` when OIDC is not configured (fx optional dependency pattern). The handler checks `oidcSvc != nil && cfg.OIDCAutoRedirect` before attempting the redirect; if either condition is false it serves `login.html` normally. This means the handler replaces the `SPAHandler`'s `/login` special-case entirely.

## Risks / Trade-offs

- **Provider discovery on first request**: `OIDCService` lazily discovers the provider on the first call. A cold-start `GET /login` with auto-redirect may be slower than normal while discovery runs. → *Mitigation*: acceptable; the `sync.Once` ensures subsequent requests are fast. Operators can warm up by hitting `/health` or another endpoint first.

- **Operator locked out**: If the OIDC provider goes down while `VYNILINO_OIDC_AUTO_REDIRECT=true`, all users are locked out. → *By design*: this is the operator's choice. The admin must either restore the provider or redeploy with the flag unset.

- **No user-visible error detail**: The 500 page gives no technical detail to the end user. → *Intentional*: error details belong in server logs, not in the browser.

## Migration Plan

1. Add `OIDCAutoRedirect bool` to `config.Config` and `Load()`.
2. Implement `loginRedirectHandler(cfg, oidcSvc, staticFS)` — a thin `http.HandlerFunc`.
3. In `server.go`, register `r.Get("/login", loginRedirectHandler(...))` before `r.Mount("/", uiHandler.SPAHandler())`.
4. Rollback: unset `VYNILINO_OIDC_AUTO_REDIRECT` — behavior reverts without a code change.
