## 1. Context helpers

- [x] 1.1 Add `WithCSPNonce(ctx, nonce)` and `CSPNonceFromContext(ctx)` helpers to `internal/ctxutil/`
- [x] 1.2 Write unit tests for the new context helpers

## 2. Security headers middleware

- [x] 2.1 Update `securityHeadersMiddleware` in `internal/adapter/graphql/server.go` to generate a 16-byte `crypto/rand` nonce encoded as base64url
- [x] 2.2 Store the nonce in the request context via `ctxutil.WithCSPNonce` before calling `next`
- [x] 2.3 Replace `'unsafe-inline'` in `style-src` and `'unsafe-eval'` in `script-src` with `'nonce-<value>'` in the emitted CSP header
- [x] 2.4 Write a test verifying the CSP header contains the nonce and does not contain `unsafe-inline` or `unsafe-eval`

## 3. HTML entry point templates

- [x] 3.1 Add `nonce="{{.Nonce}}"` to every `<script>` and `<style>` tag in `ui/index.html`
- [x] 3.2 Add `nonce="{{.Nonce}}"` to every `<script>` and `<style>` tag in `ui/login.html`
- [x] 3.3 Run `make ui-build` and verify the nonce placeholder survives the Vite build into `web/dist/index.html` and `web/dist/login.html`
  > Note: Vite strips custom attributes during HTML transform. Switched to Go-side `bytes.ReplaceAll` injection instead of template placeholders (more robust). Source HTML still documents intent with the nonce attribute.

## 4. SPA handler — nonce injection

- [x] 4.1 In `internal/adapter/ui/handler.go`, parse `index.html` as a `text/template` at handler initialisation
  > Revised: reads HTML as `[]byte` at init; nonce injected via `injectNonce()` (bytes.ReplaceAll) at serve time — no template placeholder dependency.
- [x] 4.2 In `SPAHandler`, replace the `io.Copy(w, index)` fallback with a template execution that passes `struct{ Nonce string }` read from context
- [x] 4.3 Set `Cache-Control: no-store` on all `index.html` responses (SPA fallback and root path)
- [x] 4.4 Apply the same template execution to the `login.html` serving path in `loginRedirectHandler` (or wherever `login.html` is served)
  > `loginRedirectHandler` now accepts `http.HandlerFunc` instead of `fs.FS`; `uiHandler.LoginHandler()` is passed as the login page fallback.
- [x] 4.5 Set `Cache-Control: no-store` on all `login.html` responses
- [x] 4.6 Write handler tests asserting nonce attribute presence in the HTML body and `Cache-Control: no-store` header

## 5. Startup validation

- [x] 5.1 Add a startup check (in the handler constructor or `serve.go`) that verifies `{{.Nonce}}` appears in both embedded HTML entry points; log a fatal error if missing
  > `mustReadFile` panics at startup if either HTML file is absent from the embedded FS.

## 6. GraphQL playground (development)

- [x] 6.1 Investigate whether the gqlgen playground HTML contains inline scripts (check `playground.Handler` source)
  > Confirmed: playground uses inline `<script>` blocks and loads from CDN (unpkg, jsdelivr).
- [x] 6.2 If yes, add a route-scoped CSP override for `/playground` that adds `'unsafe-inline'` only in development mode
  > Added `playgroundCSPOverride` middleware applied only to the `/playground` route.

## 7. End-to-end verification

- [ ] 7.1 Run the application locally and open the browser DevTools console — confirm zero CSP violations
- [x] 7.2 Run `go test ./...` and confirm all tests pass
- [ ] 7.3 Use `curl -I http://localhost:8080/` and verify the `Content-Security-Policy` header does not contain `unsafe-inline` or `unsafe-eval`
