## Context

The current CSP header (set in `securityHeadersMiddleware` in `internal/adapter/graphql/server.go:187`) contains:

```
script-src 'self' 'unsafe-eval'
style-src  'self' 'unsafe-inline'
```

These directives allow any inline script or style to execute, which defeats most of CSP's XSS-protection value. The fix is to issue a per-request random nonce and inject it into the HTML shell so the browser only executes scripts/styles that carry that nonce.

The UI is Alpine.js v3 + Tailwind. Alpine v3 does not require `'unsafe-eval'` for normal operation (expression evaluation was moved to a non-eval path in v3). Tailwind's output is a pre-compiled static stylesheet — no runtime `<style>` injection — so `'unsafe-inline'` for styles can be dropped entirely once the nonce is in the `<link>` tag or the CSS is served as an external file (which Vite already does).

## Goals / Non-Goals

**Goals:**
- Remove `'unsafe-inline'` from `style-src`.
- Remove `'unsafe-eval'` from `script-src`.
- Add `'nonce-<per-request>'` to both directives.
- Inject the nonce attribute into every `<script>` and `<style>` tag in `index.html` and `login.html` at serve time.
- No new runtime dependencies.

**Non-Goals:**
- Implementing Subresource Integrity (SRI) hashes for third-party CDN resources — the app has no CDN dependencies.
- Changing the CSP for `/graphql` (API-only endpoint; no HTML response).
- Modifying the GraphQL playground CSP (it is only enabled in development and uses its own static HTML — a hash approach is acceptable there, or it can keep `'unsafe-inline'` scoped to dev only).
- Dropping `'self'` from any directive.

## Decisions

### Decision 1: Nonce over hash for HTML pages

**Chosen**: Per-request random nonce injected into HTML at serve time.

**Alternatives considered**:
- **Static hash per script/style**: Requires re-computing hashes on every build and storing them (or computing at startup from the embedded FS). Works well for unchanging assets but is brittle during iterative UI development.
- **`'strict-dynamic'`**: Useful when scripts load other scripts dynamically; Alpine.js does not do this, so the added complexity is unwarranted.

**Rationale**: A per-request nonce is the standard recommendation (Google CSP Evaluator, MDN) for server-rendered or SPA pages that serve a templated HTML shell. It requires only one Go middleware change and one HTML template change per entry point.

### Decision 2: Nonce stored in request context

The nonce is generated in `securityHeadersMiddleware` (which already runs for every request) and stored via a new `ctxutil.WithCSPNonce` / `ctxutil.CSPNonceFromContext` pair, consistent with how `ResponseWriter` and `UserID` are stored today.

The SPA handler reads the nonce from context when templating `index.html` / `login.html`.

### Decision 3: Minimal Go `text/template` for nonce injection

The HTML files use a single `{{.Nonce}}` placeholder on `<script>` and `<style>` tags:

```html
<script nonce="{{.Nonce}}" src="/assets/main.js"></script>
```

At serve time, `ui/handler.go` parses the embedded HTML file as a `text/template` and executes it with `struct{ Nonce string }`. This is a one-shot in-memory render — no caching of the parsed template is needed beyond startup (the FS is embedded and static).

**Alternative considered**: String replacement (`strings.Replace`) of a sentinel value. Rejected because `text/template` is standard, handles escaping automatically, and is already a stdlib dependency.

### Decision 4: No `'unsafe-eval'` even in development

Alpine v3 uses `Function()` constructor internally only in its legacy compat build. The default ESM build does not. Vite's dev server injects an HMR client script that does use `eval` in dev mode — so in development, we add `'unsafe-eval'` back only to the dev server's Vite HMR bootstrap. In production this is irrelevant because there is no HMR code.

To keep server-side CSP consistent, the Go `securityHeadersMiddleware` will NOT special-case dev vs prod for `'unsafe-eval'`. Instead the Vite build configuration will be checked to confirm no eval usage in the output bundles.

### Decision 5: Nonce generation

16 random bytes from `crypto/rand`, encoded as base64url (no padding). This yields a 22-character nonce with ~128 bits of entropy, well above the minimum recommended 128 bits.

```go
b := make([]byte, 16)
_, _ = rand.Read(b)
nonce := base64.RawURLEncoding.EncodeToString(b)
```

## Risks / Trade-offs

- **Cached HTML responses**: If any layer (reverse proxy, CDN) caches the HTML shell, all clients will share the same nonce, defeating the protection. Mitigation: the SPA handler must set `Cache-Control: no-store` on `index.html` and `login.html` responses (already implied by the SPA fallback path, which currently sets no cache header — this PR will make it explicit).
- **Template parse error at startup**: If the embedded HTML does not contain `{{.Nonce}}` (e.g. someone edited the file without updating the placeholder), the template will execute silently with an empty nonce. Mitigation: add a startup check that verifies the placeholder exists in each entry-point HTML.
- **Alpine.js version drift**: A future Alpine version that re-introduces `eval` would break silently. Mitigation: pin Alpine to a minor version and include a note in the upgrade runbook.
- **`data:` and `blob:` in `img-src`**: Not affected by this change; they remain.

## Migration Plan

1. Update `ui/index.html` and `ui/login.html` in the Vite source to add `nonce="{{.Nonce}}"` on all `<script>` and `<style>` tags.
2. Add `ctxutil.WithCSPNonce` / `ctxutil.CSPNonceFromContext` helpers.
3. Update `securityHeadersMiddleware` to generate the nonce, store it in context, and emit the updated CSP header.
4. Update `ui/handler.go` `SPAHandler` to template-render the HTML entry points instead of `io.Copy`.
5. Run `make build` (which runs `ui-build` first) to regenerate `web/dist/`.
6. Verify with browser DevTools that the nonce attribute appears on script tags and that no CSP violations are reported in the console.

**Rollback**: Revert the CSP header string to the previous value in `securityHeadersMiddleware`. No database or migration changes are involved.

## Open Questions

- Does the GraphQL playground HTML (served by gqlgen's `playground.Handler`) contain inline scripts? If yes, a hash-based allowlist for the playground route (or a route-specific CSP override) will be needed. This should be verified during implementation.
