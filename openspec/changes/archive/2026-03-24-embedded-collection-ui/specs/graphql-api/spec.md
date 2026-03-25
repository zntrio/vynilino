## ADDED Requirements

### Requirement: Static asset serving and SPA fallback
The system SHALL serve the embedded web UI static assets and fall back to `index.html` for all non-API, non-asset GET requests, enabling client-side SPA routing.

#### Scenario: Static files served from embed.FS
- **WHEN** a GET request matches a file path present in the embedded `ui/dist/` directory
- **THEN** the system SHALL serve that file with appropriate `Content-Type` and cache headers

#### Scenario: SPA fallback does not intercept API routes
- **WHEN** a GET request path begins with `/graphql`, `/api/`, or `/auth/`
- **THEN** the system SHALL NOT apply the SPA fallback and SHALL route to the appropriate handler

#### Scenario: SPA fallback for unknown paths
- **WHEN** a GET request path does not match any registered route or static asset
- **THEN** the system SHALL return `200 OK` with `ui/dist/index.html`
