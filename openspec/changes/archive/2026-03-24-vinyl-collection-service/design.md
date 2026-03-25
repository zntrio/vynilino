## Context

Vynilino is a greenfield Go service for managing personal vinyl record collections. It targets self-hosters who want full control of their data. The service must be deployable with a single `docker compose up` and serve a hybrid Desktop/Mobile SPA via a GraphQL API. Privacy and operational simplicity are first-class constraints.

## Goals / Non-Goals

**Goals:**
- Single binary Go service with embedded migrations
- GraphQL API (queries, mutations, subscriptions) over HTTP/WebSocket
- JWT authentication with configurable single-owner or multi-user mode
- SQLite as default storage (zero external dependencies); PostgreSQL as opt-in
- Local file storage for cover art; configurable path
- Docker Compose deployment with health checks and volume mounts
- Data export (JSON/CSV) and import (Discogs CSV, generic CSV)
- No telemetry, no external calls at runtime

**Non-Goals:**
- Native mobile or desktop app (served by separate SPA)
- Multi-tenant SaaS or shared hosting
- Real-time sync between multiple server instances
- Audio playback or streaming
- Marketplace / trading features

## Decisions

### 1. GraphQL over REST
**Decision**: Use GraphQL (gqlgen) as the sole API layer.
**Rationale**: The SPA is the only client; GraphQL gives it precise data fetching, strong typing via generated schema, and subscription support for live updates. REST would require versioned endpoints and over/under-fetch workarounds.
**Alternative considered**: REST + SSE — simpler but loses type safety and introspection value.

### 2. SQLite default with PostgreSQL option
**Decision**: Default to SQLite via `modernc.org/sqlite` (CGo-free); support PostgreSQL via build tag or env var.
**Rationale**: SQLite eliminates the need for a separate database container in the default self-hosted setup. A single volume mount backs up everything. Power users who need concurrency or replication can switch to PostgreSQL.
**Alternative considered**: PostgreSQL-only — too heavy for a simple home server.

### 3. sqlc for database layer
**Decision**: Use `sqlc` for type-safe SQL query generation.
**Rationale**: Avoids ORM magic while keeping SQL explicit and readable. Generated code is auditable and type-safe. Works with both SQLite and PostgreSQL dialects.
**Alternative considered**: GORM — too much hidden complexity; Bun — good but sqlc keeps SQL visible.

### 4. golang-migrate for schema migrations
**Decision**: Embed migrations in the binary using `embed.FS`; run on startup with `golang-migrate`.
**Rationale**: Zero-touch deployments. No separate migration step; operators just restart the container.

### 5. JWT authentication (paseto v2 local)
**Decision**: Use PASETO v2 local tokens (symmetric) for auth tokens instead of JWT.
**Rationale**: PASETO avoids JWT's algorithm confusion vulnerabilities and `alg: none` attacks. Simpler secret management for self-hosters (one symmetric key).
**Alternative considered**: Standard JWT (HMAC-SHA256) — acceptable but PASETO is strictly safer with no added complexity.

### 6. Chi router + gqlgen
**Decision**: Use `go-chi/chi` as HTTP router wrapping the gqlgen GraphQL handler.
**Rationale**: Lightweight, idiomatic, good middleware ecosystem. gqlgen generates type-safe resolvers from the schema, keeping the schema as the source of truth.

### 7. Hexagonal architecture (ports & adapters)
**Decision**: Structure the codebase with a clear domain core, ports (interfaces), and adapters (implementations).
**Rationale**: Keeps business logic testable without infrastructure. Allows swapping SQLite → PostgreSQL or local storage → S3 without touching domain code.

```
cmd/server/          — entrypoint, wiring
internal/
  domain/            — entities, value objects, repository interfaces
  app/               — use cases / application services
  adapter/
    graphql/         — gqlgen resolvers (HTTP adapter)
    storage/sqlite/  — SQLite repository implementations
    storage/postgres/ — PostgreSQL repository implementations
    filestore/       — local file storage adapter
  config/            — configuration loading (env vars + YAML)
```

### 8. Cover art storage
**Decision**: Store cover art as files on disk (configurable `VYNILINO_MEDIA_DIR`); serve via authenticated HTTP endpoint outside GraphQL.
**Rationale**: Binary blobs in the database are bad for SQLite performance and backup size. A simple static file server with auth middleware is sufficient.

## Risks / Trade-offs

- **SQLite write concurrency** → Mitigation: Enable WAL mode; document that concurrent writes are limited; recommend PostgreSQL for multi-user setups.
- **PASETO key rotation** → Mitigation: Document key rotation procedure; support `VYNILINO_TOKEN_KEY` env var rotation with a brief dual-validation window.
- **GraphQL N+1 queries** → Mitigation: Use `graph-gophers/dataloader` for batching in resolvers.
- **File storage not backed up automatically** → Mitigation: Document backup strategy; provide example `restic` / `rclone` snippets in docs.
- **Single binary includes migrations** → If migration fails on startup, service won't start. Mitigation: Dry-run migration check flag (`--check-migrations`); clear error messages.

## Migration Plan

This is a greenfield project — no existing data to migrate.

Deployment sequence:
1. Build Docker image: `docker build -t vynilino .`
2. Configure `.env` (or env vars): `VYNILINO_TOKEN_KEY`, `VYNILINO_DB_PATH`, `VYNILINO_MEDIA_DIR`
3. Run: `docker compose up -d`
4. Service auto-runs migrations on first start
5. Register first user (single-owner mode auto-promotes to admin)

Rollback: stop container, restore volume backup, start previous image version.

## Open Questions

- Should subscriptions (live collection updates) be in scope for v1, or deferred to v2?
- Discogs API integration (fetching metadata automatically) — v1 import-only or also live lookup?
- Should cover art support external URLs (e.g., MusicBrainz cover art archive) in addition to uploads?
