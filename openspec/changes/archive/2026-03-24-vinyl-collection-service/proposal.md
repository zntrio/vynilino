## Why

Vinyl record collectors lack a self-hostable, privacy-respecting tool to manage their collections. Existing solutions are cloud-dependent, opaque about data usage, and offer no API for custom clients. This project fills that gap with a Go service exposing a GraphQL API designed for hybrid Desktop/Mobile SPA consumption.

## What Changes

- Introduce a new Go-based backend service (`vynilino`) for vinyl record collection management
- Expose a GraphQL API as the sole interface for client applications
- Provide Docker-based self-hosting with minimal operational overhead
- Implement authentication and authorization with strong privacy defaults
- Store all data locally (SQLite by default, PostgreSQL optional) with no external telemetry

## Capabilities

### New Capabilities

- `collection-management`: CRUD operations for vinyl records — add, edit, remove, and list records with metadata (artist, album, year, label, condition, notes, cover art)
- `search-and-filter`: Full-text search and filtering across the collection by artist, album, genre, year, condition
- `user-auth`: JWT-based authentication with registration, login, token refresh, and logout; single-owner mode for self-hosted deployments
- `graphql-api`: GraphQL schema, resolvers, and transport layer (HTTP + WebSocket for subscriptions) serving the SPA client
- `media-management`: Cover art upload, storage, and serving with configurable local storage backend
- `data-portability`: Export collection as CSV/JSON and import from Discogs/CSV for data sovereignty

### Modified Capabilities

<!-- none — this is a greenfield project -->

## Impact

- New Go module at repository root (`cmd/server`, `internal/...`)
- GraphQL schema defines the public API contract for all SPA clients
- SQLite default eliminates external database dependency for simple self-hosting
- No third-party analytics, telemetry, or cloud dependencies by default
- Docker Compose file provided for one-command deployment
