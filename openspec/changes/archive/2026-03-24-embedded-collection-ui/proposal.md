## Why

Vynilino has a fully-featured GraphQL backend but no user-facing interface. Vinyl collectors need a fast, responsive UI to browse, add, edit, and search their records from both desktop and mobile devices without installing a native app.

## What Changes

- Introduce a new embedded web UI served directly by the Go backend at `GET /` (and sub-routes)
- The UI provides full collection management: list, add, edit, delete records with cover art
- Responsive layout adapts between desktop sidebar-based navigation and mobile bottom-nav
- Authentication flow integrates with the existing OIDC provider (redirect-based login)
- Real-time updates via the existing GraphQL WebSocket subscriptions
- Static assets are bundled and embedded into the Go binary at build time (no separate CDN needed)

## Capabilities

### New Capabilities

- `collection-ui`: Embedded hybrid web interface for managing the vinyl collection — covers UI routing, collection CRUD views, search/filter panel, media upload, responsive layout, and OIDC login flow

### Modified Capabilities

- `graphql-api`: Add static asset serving and SPA fallback route (`GET /*`) to the existing HTTP server so the UI is served from the same origin as the API

## Impact

- **Frontend**: New `ui/` directory with a lightweight framework (Alpine.js + HTMX or SvelteKit slim build) and Tailwind CSS for styling
- **Backend**: Add `net/http` static file handler and embed directive; no schema changes
- **Build**: New `make ui` step to compile frontend assets before Go build
- **Dependencies**: Minimal — no React/Vue/Angular; target < 50 kB gzipped JS payload
