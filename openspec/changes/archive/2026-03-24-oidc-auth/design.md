## Context

vynilino's authentication is currently local-only: email + Argon2id password, PASETO v4 local access tokens, rotating refresh tokens in SQLite. The `UserService` owns the full auth lifecycle. There is no federation, SSO, or external identity support.

OIDC (OpenID Connect 1.0) is the standard protocol for delegated authentication. We want to add it as an optional, additive auth path that does not break or change the existing local flow.

Key constraints:
- Must stay CGo-free (pure-Go OIDC library required)
- OIDC must be opt-in; if `VYNILINO_OIDC_ISSUER` is unset the feature is entirely absent at runtime
- Issued tokens must remain PASETO v4 local (OIDC tokens are consumed internally only)
- Single-owner mode still applies: OIDC auto-provisioning is blocked if an account already exists
- Must support Authorization Code Flow with PKCE (S256); implicit flow is not supported

## Goals / Non-Goals

**Goals:**
- Authorization Code Flow + PKCE via provider discovery (`.well-known/openid-configuration`)
- Auto-provision a vynilino user on first OIDC login (`sub` claim as stable identity key)
- Link OIDC identity to an existing local account (by matching `email` claim)
- Issue vynilino PASETO access + refresh tokens after successful OIDC authentication
- New GraphQL mutations: `oidcAuthorizationURL` and `oidcCallback`
- State/nonce stored server-side (short-lived, in-memory or SQLite) to prevent CSRF/replay
- Runtime configuration via four env vars; no code changes to enable/disable

**Non-Goals:**
- Multiple simultaneous OIDC providers (single provider only in this iteration)
- Back-channel logout / token introspection
- OIDC for API-to-API (machine credentials); only user-facing flows
- Changing existing password-based auth in any way
- Social login UX (that's the SPA's responsibility)

## Decisions

### D1: Pure-Go OIDC library — `coreos/go-oidc/v3`

**Choice**: `github.com/coreos/go-oidc/v3` + `golang.org/x/oauth2`

**Rationale**: The most widely adopted pure-Go OIDC client. Handles provider discovery, JWKS caching, ID token verification (signature, `iss`, `aud`, `exp`, nonce). No CGo. Works with any standards-compliant IdP. Alternative `zitadel/oidc` is full server+client and would pull unnecessary dependencies; `dex/connector` is server-side only.

### D2: State/nonce stored in SQLite, not in-memory

**Choice**: New `oidc_states` table with `state` (random 32-byte hex), `nonce` (random 32-byte hex), `code_verifier`, `created_at`, TTL enforced at read time (5 minutes).

**Rationale**: In-memory state survives only within one process instance. A single-process self-hosted deployment could work with in-memory, but a restart between `oidcAuthorizationURL` and `oidcCallback` would lose state. SQLite is already available; the cost is one small table. State entries are deleted on successful callback or on expiry.

### D3: Account linking strategy — `email` claim + `sub` claim

**Choice**: On first OIDC login:
1. Look up `oidc_identities` by `(provider, sub)` — if found, use that `user_id`.
2. If not found, look up users by `email` claim — if found, create an `oidc_identities` row linking that user (account linking).
3. If no user exists, auto-provision: create a new `users` row (no password), create an `oidc_identities` row.

Single-owner mode check: step 3 is blocked if any user already exists.

**Rationale**: `sub` is stable and provider-scoped; `email` linkage handles the common self-hosters case of registering with local account first, then wanting to switch to OIDC. Alternative (require explicit link mutation) is more secure but creates friction; for a personal self-hosted app the risk is acceptable.

### D4: OIDCService separate from UserService

**Choice**: New `app.OIDCService` struct with its own constructor, injected via uber/fx.

**Rationale**: Keeps `UserService` focused on local-credential flows. `OIDCService` depends on `UserService` (to issue tokens after linking/provisioning) and on new `OIDCStateRepository` / `OIDCIdentityRepository`. Avoids turning `UserService` into a god object. Alternative (methods on `UserService`) blurs concerns and grows the already-large service.

### D5: PKCE with S256 mandatory

**Choice**: Always generate `code_verifier` (43-128 random chars), store with state, send `code_challenge=BASE64URL(SHA256(verifier))` with `code_challenge_method=S256`.

**Rationale**: PKCE is required by OAuth 2.1 and recommended by current BCP. Some older IdPs don't support it; they are out of scope. Plain (no PKCE) is not offered. This avoids the authorization code interception attack and is standard practice.

## Risks / Trade-offs

| Risk | Mitigation |
|------|-----------|
| Provider discovery fails at startup | Discovery is deferred to first request, not startup; config validation checks issuer URL format only. A failed discovery returns a clear error to the client. |
| `email` claim absent or unverified | Only link by email if the `email_verified` claim is `true`; otherwise treat as new user (provision without linking). |
| State table grows unboundedly | Background cleanup: prune entries older than 10 minutes on each `oidcAuthorizationURL` call (lazy GC, acceptable for single-user). |
| CSRF via state forgery | State is 32 random bytes (256-bit entropy); verified server-side before any token exchange. |
| Account takeover via unverified email claim | Mitigated by `email_verified` check in D3. |
| IdP outage makes login impossible | Local email/password flow is unaffected; OIDC is additive only. |

## Migration Plan

1. Add new migration `000002_oidc.up.sql` (new tables: `oidc_identities`, `oidc_states`).
2. New tables are purely additive; existing data is untouched. No down-time required.
3. Rollback: `000002_oidc.down.sql` drops the two new tables. Local auth is unaffected.
4. OIDC is disabled at runtime unless `VYNILINO_OIDC_ISSUER` is set; deploy without it for a no-op upgrade.

## Open Questions

- Should `oidcAuthorizationURL` be authenticated (require a logged-in user for explicit linking) or unauthenticated (for new user login)? → Unauthenticated; linking to an existing account happens transparently via email claim match on callback.
- Should the `sub` claim be stored hashed or plaintext? → Plaintext; it is an opaque public identifier from the IdP, not a secret.
