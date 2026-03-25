## MODIFIED Requirements

### Requirement: Serve embedded web UI
The system SHALL serve two separate single-page application shells from static assets embedded in the Go binary at build time via `//go:embed ui/dist`.

- `GET /login` (and `/login.html`) SHALL serve `ui/dist/login.html` — the login-only bundle.
- `GET /` and all non-API, non-asset, non-login sub-paths SHALL serve `ui/dist/index.html` — the authenticated app shell.

When serving `index.html` or `login.html`, the system SHALL perform template substitution to inject the per-request CSP nonce into all `<script>` and `<style>` tags before writing the response body. The HTML source files SHALL contain `nonce="{{.Nonce}}"` on those tags as the substitution placeholder.

Responses serving `index.html` or `login.html` SHALL include `Cache-Control: no-store`.

#### Scenario: Root path returns index.html
- **WHEN** an authenticated browser requests `GET /`
- **THEN** the system SHALL return `200 OK` with `Content-Type: text/html` and the contents of `ui/dist/index.html` with the nonce injected

#### Scenario: Login path returns login.html
- **WHEN** any browser requests `GET /login` or `GET /login.html`
- **THEN** the system SHALL return `200 OK` with `Content-Type: text/html` and the contents of `ui/dist/login.html` with the nonce injected

#### Scenario: Static assets served with cache headers
- **WHEN** a browser requests a hashed asset such as `GET /assets/main.abc123.js`
- **THEN** the system SHALL return `200 OK` with `Cache-Control: public, max-age=31536000, immutable`

#### Scenario: SPA fallback for deep links (authenticated app)
- **WHEN** a browser requests any path under `/` that is not `/graphql`, `/api/`, `/auth/`, `/login`, `/login.html`, or a known static asset
- **THEN** the system SHALL return `200 OK` with the nonce-injected contents of `ui/dist/index.html` to support client-side routing

#### Scenario: HTML entry point response is not cached
- **WHEN** the server serves `index.html` or `login.html`
- **THEN** the response SHALL include `Cache-Control: no-store`
