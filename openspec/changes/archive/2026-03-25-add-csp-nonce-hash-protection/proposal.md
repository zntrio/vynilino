## Why

The current `Content-Security-Policy` header allows `'unsafe-eval'` for scripts and `'unsafe-inline'` for styles, which undermines XSS protection by permitting arbitrary inline execution. Replacing these broad permissions with per-request nonces and/or static hashes eliminates the largest attack surface in our CSP without breaking the Alpine.js + Tailwind UI.

## What Changes

- Remove `'unsafe-inline'` and `'unsafe-eval'` from the CSP `script-src` and `style-src` directives.
- Generate a cryptographically random nonce per HTTP request in `securityHeadersMiddleware`.
- Store the nonce in the request context so it can be injected into HTML responses.
- Modify the SPA handler (`ui/handler.go`) to template `index.html` with the nonce, inserting `nonce="<value>"` on every `<script>` and `<style>` tag.
- Update CSP header to `script-src 'self' 'nonce-<value>'` and `style-src 'self' 'nonce-<value>'`.
- For the `/playground` GraphQL playground (development only), apply a static SHA-256 hash instead of a nonce, as the playground HTML is static and served by the gqlgen library.

## Capabilities

### New Capabilities

- `csp-nonce`: Per-request CSP nonce generation, context propagation, and HTML injection for the SPA and login pages.

### Modified Capabilities

- `collection-ui`: The SPA index.html serving path gains nonce injection, changing how the HTML response is produced (no longer a raw static file copy).

## Impact

- **`internal/adapter/graphql/server.go`**: `securityHeadersMiddleware` generates and stores a nonce; CSP header updated.
- **`internal/adapter/ui/handler.go`**: `SPAHandler` and the `index.html` fallback path switch from raw `io.Copy` to a minimal template execution that injects the nonce.
- **`internal/ctxutil/`**: New context key and helper functions (`WithCSPNonce`, `CSPNonceFromContext`).
- **`web/` (UI build)**: All inline `<script>` and `<style>` tags in `index.html` and `login.html` must carry the nonce attribute placeholder (e.g., `{{.Nonce}}` or a marker replaced at serve time).
- **No new external dependencies** — uses `crypto/rand` and `encoding/base64` from the standard library.
- **Development mode**: `'unsafe-eval'` removed; Alpine.js v3 does not require it. If integration tests reveal otherwise, a hash fallback is acceptable for dev only.
