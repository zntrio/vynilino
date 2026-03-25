## Context

Vynilino is a Go 1.26 backend (chi router, gqlgen GraphQL, OIDC auth) with no frontend. The binary is self-contained and deployed as a single process. The goal is to ship a hybrid-responsive UI embedded in that same binary, so operators do not need to manage a separate CDN or SPA host.

Existing adapters: `internal/adapter/graphql` (HTTP + WS), `internal/adapter/storage` (DB), `internal/adapter/filestore` (cover art). A new `internal/adapter/ui` adapter will be added to serve static assets.

## Goals / Non-Goals

**Goals:**
- Embedded, zero-external-hosting UI served from `GET /` via Go's `embed.FS`
- Full collection CRUD (list, add, edit, delete) with cover art upload
- Responsive layout: desktop (sidebar nav) and mobile (bottom nav / drawer)
- Search & filter panel backed by existing GraphQL queries
- OIDC login/logout flow (redirect-based, same origin)
- Real-time optimistic updates via GraphQL subscriptions (WebSocket)
- < 50 kB gzipped JS payload; no build-time Node.js in production Docker image

**Non-Goals:**
- Native mobile apps (iOS / Android)
- Offline / PWA service worker (can be added later)
- Server-side rendering / hydration (too complex for the embedded constraint)
- Separate deployment of the frontend

## Decisions

### D1 — Framework: Alpine.js + HTMX over React/Vue/SvelteKit

**Choice**: Alpine.js 3.x for reactive state + HTMX 2.x for declarative server interactions, styled with Tailwind CSS 4.x (CDN-free, built via CLI).

**Rationale**: Combined gzipped size ≈ 18 kB (Alpine) + 14 kB (HTMX) + ~8 kB utility classes = ~40 kB, well under budget. No virtual DOM, no complex build pipeline. Alpine provides scoped reactivity for modals, forms, and live updates; HTMX handles navigation and partial HTML swaps. This avoids shipping a JavaScript framework runtime.

**Alternative considered**: SvelteKit — excellent DX but requires Node.js toolchain in CI and produces a heavier bundle; also needs SSR adapter changes. Rejected due to embedded complexity.

**Alternative considered**: Vanilla JS — viable but requires hand-rolling routing, reactivity, and WebSocket handling. Maintenance cost too high.

### D2 — Build pipeline: Vite (dev) + Go embed (prod)

**Choice**: Vite used only for development (HMR, Tailwind JIT). Production build outputs `ui/dist/` which is compiled into the binary via `//go:embed ui/dist`.

**Rationale**: Vite is a dev-time dependency only; the Docker image installs Node.js only during the multi-stage build step, not in the final image. The Go binary remains the only runtime artefact.

### D3 — API communication: GraphQL over HTTP + WebSocket

**Choice**: Use the existing `POST /graphql` endpoint for queries/mutations; use the existing `GET /graphql` WebSocket endpoint for real-time subscription updates on the collection list.

**Rationale**: No new API surface. The UI speaks the same protocol as any other GraphQL client, so backend changes are minimal (only static serving is added).

### D4 — Authentication: OIDC redirect flow, session cookie

**Choice**: The UI initiates login via `GET /auth/login` (existing OIDC redirect). After callback, the backend sets an `HttpOnly` session cookie. The UI reads a `GET /api/me` endpoint (new, thin) to determine auth state on load.

**Rationale**: Keeps tokens server-side. Alpine.js checks `/api/me` on boot and redirects to `/auth/login` if unauthenticated.

### D5 — Routing: SPA with Go fallback

**Choice**: All unknown `GET` paths return `index.html` (SPA fallback) from the Go router, implemented in `internal/adapter/ui`. Client-side routing uses the History API via a tiny (~1 kB) router in Alpine.

**Rationale**: Allows deep-linking to `/collection/123` without server-side route registration per page.

## Risks / Trade-offs

| Risk | Mitigation |
|---|---|
| HTMX partial-swap and Alpine reactivity overlap creates confusion | Establish a clear boundary: HTMX for navigation/list swaps, Alpine for in-page state (modals, forms, live counters) |
| Tailwind CSS build step adds CI complexity | Pin Tailwind CLI binary version; download in Makefile, no npm needed |
| WebSocket subscription re-auth on token expiry | Subscribe after `/api/me` check; reconnect with exponential back-off on 4401 close code |
| `embed.FS` increases binary size | Acceptable trade-off; cover art is stored externally (filestore), not in binary |
| SPA fallback conflicts with future REST endpoints at `/api/*` | Route all API traffic under `/api/` and `/graphql`; SPA fallback only fires for non-API paths |

## Migration Plan

1. Add `ui/` directory with Vite config, `src/` sources, and `dist/` gitignored
2. Add `internal/adapter/ui/` Go package with `embed.FS` and chi handler registration
3. Wire UI adapter into `cmd/vynilino/main.go` router after existing adapters
4. Add `make ui-build` target; update `make build` to depend on it
5. Add `GET /api/me` thin handler (returns current user from session)
6. Update Docker multi-stage build: add Node.js stage before Go build stage
7. Document dev workflow: `make ui-dev` runs Vite dev server proxied to Go backend

**Rollback**: Remove `ui/` adapter registration from router; the binary continues to work as an API-only service without UI.

## Open Questions

- Should cover art uploads go through GraphQL multipart or a dedicated `POST /api/upload` REST endpoint? (Lean toward REST for simplicity with HTMX file inputs.)
- Pagination strategy for large collections: infinite scroll vs. "load more" button? (Lean toward "load more" for accessibility.)
