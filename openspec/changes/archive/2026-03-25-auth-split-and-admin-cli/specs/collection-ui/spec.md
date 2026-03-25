## MODIFIED Requirements

### Requirement: Serve embedded web UI
The system SHALL serve two separate single-page application shells from static assets embedded in the Go binary at build time via `//go:embed ui/dist`.

- `GET /login` (and `/login.html`) SHALL serve `ui/dist/login.html` — the login-only bundle.
- `GET /` and all non-API, non-asset, non-login sub-paths SHALL serve `ui/dist/index.html` — the authenticated app shell.

#### Scenario: Root path returns index.html
- **WHEN** an authenticated browser requests `GET /`
- **THEN** the system SHALL return `200 OK` with `Content-Type: text/html` and the contents of `ui/dist/index.html`

#### Scenario: Login path returns login.html
- **WHEN** any browser requests `GET /login` or `GET /login.html`
- **THEN** the system SHALL return `200 OK` with `Content-Type: text/html` and the contents of `ui/dist/login.html`

#### Scenario: Static assets served with cache headers
- **WHEN** a browser requests a hashed asset such as `GET /assets/main.abc123.js`
- **THEN** the system SHALL return `200 OK` with `Cache-Control: public, max-age=31536000, immutable`

#### Scenario: SPA fallback for deep links (authenticated app)
- **WHEN** a browser requests any path under `/` that is not `/graphql`, `/api/`, `/auth/`, `/login`, `/login.html`, or a known static asset
- **THEN** the system SHALL return `200 OK` with the contents of `ui/dist/index.html` to support client-side routing

## ADDED Requirements

### Requirement: Login bundle isolation
The Vite build SHALL produce two independent entry points with no shared JavaScript chunks between them:

- `login.html` + `login.js`: loads Alpine.js, the login form component, and OIDC redirect logic only.
- `index.html` + `main.js`: loads the full application (Alpine.js, router, all views).

Neither bundle SHALL import or statically depend on modules from the other entry point.

#### Scenario: Login page loads without application code
- **WHEN** an unauthenticated user navigates to `/login`
- **THEN** the browser SHALL download only the login bundle assets (login.js and its dependencies) and SHALL NOT download main.js or any app-only chunk

#### Scenario: App page loads without login code
- **WHEN** an authenticated user navigates to `/`
- **THEN** the browser SHALL download only the app bundle assets and SHALL NOT download login.js

### Requirement: Login view standalone entry point
The login view (`src/login.js`) SHALL be a self-contained Alpine.js application that:
- Initialises Alpine with only the auth store and toast store.
- Renders the login form (email/password) with submit triggering the GraphQL `login` mutation.
- Renders the OIDC login button (if OIDC is available) triggering the `oidcAuthorizationURL` mutation and redirecting.
- On successful login, stores the access token and redirects to `/`.

#### Scenario: Successful password login redirects to app
- **WHEN** a user submits valid credentials on the login page
- **THEN** the login bundle SHALL store the token in localStorage and navigate the browser to `/`

#### Scenario: Failed login shows error
- **WHEN** a user submits invalid credentials
- **THEN** the login bundle SHALL display an inline error message without redirecting
