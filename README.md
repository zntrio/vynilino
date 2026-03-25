# vynilino

[![SLSA Level 2](https://slsa.dev/images/gh-badge-level2.svg)](SECURITY.md)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

A self-hosted vinyl record collection manager with a GraphQL API.

> **Supply chain verification**: Release container images are signed with cosign and include SLSA provenance and an SPDX SBOM. See [SECURITY.md](SECURITY.md) for verification instructions.

## About

Vynilino lets you catalogue, search, and manage your vinyl record collection from a single self-hosted instance. It connects to the [Discogs](https://www.discogs.com/) database for metadata and cover-art lookup, stores everything in a local SQLite file, and exposes a GraphQL API consumed by a lightweight built-in web UI.

Key properties:

- **Self-hosted first** — no cloud account required; all data stays on your server.
- **Single binary** — the Go backend embeds the UI assets; one file to deploy.
- **Lightweight** — < 30 kB gzipped JS bundle (Alpine.js + Tailwind CSS 4); no React/Vue runtime.
- **OIDC-ready** — integrate with any standards-compliant identity provider.
- **Supply-chain secure** — SLSA Level 2 provenance, keyless cosign signatures, and SPDX SBOM on every release.

## How this project was built — vibe engineering with OpenSpec

Vynilino was **vibe engineered**: the entire codebase was designed and implemented through an AI-assisted, spec-driven workflow powered by [OpenSpec](https://openspec.dev/).

### What is vibe engineering?

Vibe engineering is a development approach where a human collaborates with an AI assistant (here, Claude Code) to go from idea to working software. The human describes intent and reviews outcomes; the AI writes code, runs tests, and iterates. The key discipline that makes this work at scale is **spec-driven development**.

### What is OpenSpec?

OpenSpec is a lightweight, file-based specification system that lives inside the repository (`openspec/`). Each feature starts as a **proposal** → **design** → **specs** → **tasks** chain. The AI reads these documents, implements the tasks, and marks them complete. The result is:

- A full audit trail of every design decision inside the repo.
- Human-readable specs that explain *why* code exists, not just *what* it does.
- A repeatable workflow that a new contributor (or AI session) can pick up at any point.

### Repository layout for OpenSpec artifacts

```
openspec/
  config.yaml               # project-level OpenSpec config
  specs/                    # current, living specifications per domain
  changes/
    archive/                # completed changes, one directory per change
      YYYY-MM-DD-<slug>/
        .openspec.yaml      # change metadata
        proposal.md         # what and why
        design.md           # how (architecture, trade-offs)
        specs/              # per-domain spec deltas for this change
        tasks.md            # implementation checklist
```

Archived changes are the project's decision log. Reading them in chronological order tells the full story of how the project evolved.

## Quickstart (Docker Compose)

```bash
# 1. Generate a 32-byte symmetric key (hex-encoded)
export VYNILINO_TOKEN_KEY=$(openssl rand -hex 32)

# 2. Set allowed origins for your SPA
export VYNILINO_ALLOWED_ORIGINS="http://localhost:3000"

# 3. Start the service
docker compose up -d
```

The service is now available at `http://localhost:8080`.

## Reverse Proxy / TLS Termination (Caddy)

Vynilino binds to plain HTTP on `:8080`. For production, put [Caddy](https://caddyserver.com/) in front to handle TLS termination and automatic certificate management via Let's Encrypt.

### Public domain (automatic HTTPS)

```caddyfile
vinyl.example.com {
    reverse_proxy localhost:8080 {
        header_up X-Real-IP        {remote_host}
        header_up X-Forwarded-For  {remote_host}
        header_up X-Forwarded-Proto {scheme}
    }

    header {
        Strict-Transport-Security "max-age=31536000; includeSubDomains; preload"
        X-Content-Type-Options    "nosniff"
        X-Frame-Options           "DENY"
        Referrer-Policy           "strict-origin-when-cross-origin"
        -Server
    }

    encode gzip
}
```

Caddy proxies WebSocket upgrades (GraphQL subscriptions) automatically — no extra configuration needed.

Update these environment variables to match your public domain:

```bash
VYNILINO_OIDC_REDIRECT_URL=https://vinyl.example.com/oidc/callback
VYNILINO_ALLOWED_ORIGINS=https://vinyl.example.com
```

### Local / self-signed TLS

Use Caddy's built-in CA for a locally-trusted certificate (useful for LAN deployments or development over HTTPS):

```caddyfile
vinyl.local {
    tls internal
    reverse_proxy localhost:8080
}
```

### Custom certificate

```caddyfile
vinyl.example.com {
    tls /path/to/cert.pem /path/to/key.pem
    reverse_proxy localhost:8080
}
```

## Environment Variables

| Variable                      | Default                            | Description                                      |
|-------------------------------|------------------------------------|--------------------------------------------------|
| `VYNILINO_LISTEN_ADDR`        | `:8080`                            | TCP address to bind                              |
| `VYNILINO_ENV`                | `production`                       | `development` enables playground + verbose logs  |
| `VYNILINO_DB_PATH`            | `./data/vynilino.db`               | SQLite database file path                        |
| `VYNILINO_MEDIA_DIR`          | `./data/media`                     | Directory for cover art files                    |
| `VYNILINO_TOKEN_KEY`          | *(required)*                       | 32-byte hex-encoded PASETO symmetric key         |
| `VYNILINO_TOKEN_KEY_NEW`      | *(unset)*                          | New key during rotation bridge period; see [key rotation runbook](docs/runbooks/key-rotation.md) |
| `VYNILINO_SINGLE_OWNER`       | `true`                             | Only one user account allowed when `true`        |
| `VYNILINO_BOOTSTRAP_TOKEN`    | *(unset)*                          | When set, required as a one-time token for first-user registration |
| `VYNILINO_ALLOWED_ORIGINS`    | `http://localhost:3000,...`        | Comma-separated CORS allowed origins             |
| `VYNILINO_PLAYGROUND`         | `false` (`true` in development)    | Serve GraphQL Playground at `/playground`        |
| `VYNILINO_INTROSPECTION`      | `false` (`true` in development)    | Enable GraphQL introspection                     |
| `VYNILINO_OIDC_ISSUER`        | *(unset — OIDC disabled)*          | OIDC provider issuer URL (enables OIDC when set) |
| `VYNILINO_OIDC_CLIENT_ID`     | *(unset)*                          | OIDC application client ID                       |
| `VYNILINO_OIDC_CLIENT_SECRET` | *(unset)*                          | OIDC application client secret                   |
| `VYNILINO_OIDC_REDIRECT_URL`  | *(unset)*                          | Callback URL registered with the OIDC provider   |
| `VYNILINO_OIDC_AUTO_REDIRECT` | `false`                            | When `true`, `GET /login` redirects directly to the OIDC provider |
| `VYNILINO_DISCOGS_TOKEN`      | *(unset)*                          | Personal access token for higher Discogs API rate limits |
| `VYNILINO_BEHIND_PROXY`       | `false`                            | Trust `X-Forwarded-For` / `X-Real-IP` headers (set only behind a known reverse proxy) |
| `VYNILINO_TLS_CERT`           | *(unset)*                          | Path to TLS certificate PEM file (enables native TLS) |
| `VYNILINO_TLS_KEY`            | *(unset)*                          | Path to TLS private key PEM file                 |
| `VYNILINO_BACKUP_HMAC_KEY`    | *(unset)*                          | HMAC-SHA256 key for backup authenticity signing/verification |

## API

GraphQL endpoint: `POST /graphql`
WebSocket (subscriptions): `GET /graphql` (upgrade)

### Auth Flow

```graphql
# Register (first user becomes admin in single-owner mode)
mutation {
  register(email: "you@example.com", password: "YourPass1!") {
    accessToken
    refreshToken
    expiresIn
  }
}

# Login
mutation {
  login(email: "you@example.com", password: "YourPass1!") {
    accessToken
    refreshToken
  }
}
```

Pass `Authorization: Bearer <accessToken>` on all subsequent requests.

### Collection

```graphql
mutation {
  createRecord(input: {
    title: "The Dark Side of the Moon"
    artist: "Pink Floyd"
    year: 1973
    format: LP
    condition: NEAR_MINT
  }) {
    record { id title artist }
    duplicateWarning
  }
}

query {
  records(first: 20, filter: { artist: "Floyd" }, sort: { field: YEAR, direction: DESC }) {
    edges { node { id title year condition } }
    pageInfo { hasNextPage endCursor }
    totalCount
  }
}
```

### Cover Art

```bash
curl -X POST http://localhost:8080/media/cover-art \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@cover.jpg" \
  -F "recordId=<record-id>"
```

## Import / Export

```bash
# Export as JSON
curl -o collection.json \
  -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/export/json

# Export as CSV
curl -o collection.csv \
  -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/export/csv

# Import from CSV (including Discogs export format)
curl -X POST http://localhost:8080/import/csv \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@discogs_export.csv"
```

## Backup

All data is stored in two locations:

- **Database**: `VYNILINO_DB_PATH` (single SQLite file)
- **Cover art**: `VYNILINO_MEDIA_DIR` (flat directory per user)

### Built-in backup CLI

Vynilino ships a `backup` subcommand that creates a compact, verified snapshot using SQLite's `VACUUM INTO` and optionally signs it with HMAC-SHA256.

```bash
# Create a backup (saved next to the database file)
vynilino backup create --db ./data/vynilino.db

# Create a signed backup (recommended for tamper detection)
vynilino backup create \
  --db ./data/vynilino.db \
  --output /backups/ \
  --hmac-key "$VYNILINO_BACKUP_HMAC_KEY"

# Verify backup integrity and row count
vynilino backup verify \
  --backup /backups/vynilino-20260325-120000.db \
  --hmac-key "$VYNILINO_BACKUP_HMAC_KEY"
```

Each backup produces three files:
- `<name>-<timestamp>.db` — the compacted SQLite snapshot
- `<name>-<timestamp>.db.count` — expected row count sidecar
- `<name>-<timestamp>.db.sig` — HMAC-SHA256 signature (only when `--hmac-key` is set)

### Example backup with restic

```bash
restic backup \
  /path/to/vynilino.db \
  /path/to/media/
```

### Example backup with rclone

```bash
rclone sync /path/to/data/ remote:vynilino-backup/
```

## Migration Check

Verify that the database schema is up to date without starting the server:

```bash
docker run --rm \
  -v vynilino_data:/data \
  -e VYNILINO_DB_PATH=/data/vynilino.db \
  -e VYNILINO_TOKEN_KEY=<key> \
  vynilino -check-migrations
```

## Web UI

Vynilino ships a built-in hybrid web UI (desktop + mobile) served at `GET /`. It is embedded directly into the Go binary at build time — no separate CDN or SPA host required.

### UI Development Workflow

```bash
# Install frontend dependencies (first time only)
make ui-install

# Start the Vite dev server (hot-reload) proxied to your running Go backend
make ui-dev
# → opens http://localhost:5173

# Build production assets into web/dist/ (runs automatically before `make build`)
make ui-build
```

### Lightweight by design

The production JS bundle is **< 30 kB gzipped** using Alpine.js + Tailwind CSS 4. No React, Vue, or Angular runtime.

### REST helpers (UI-specific endpoints)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/api/me` | Bearer | Returns `{"id","email"}` for the current user or `401` |
| `POST` | `/api/upload` | Bearer | Accepts `multipart/form-data` with a `file` field (≤ 5 MB JPEG/PNG/WebP); returns `{"url":"..."}` |

### Architecture note

Static assets are compiled by Vite into `web/dist/` and embedded in the Go binary via `//go:embed all:dist` in `web/embed.go`. The `internal/adapter/ui` package exposes:

- `SPAHandler()` — serves static files with immutable cache headers; falls back to `index.html` for all non-API paths (SPA routing)
- `MeHandler()` — `GET /api/me`
- `UploadHandler()` — `POST /api/upload`

API routes (`/graphql`, `/api/`, `/auth/`, `/media/`, `/export/`, `/import/`) are registered before the SPA handler and always take precedence.

## Development

```bash
# Copy env template
cp .env.example .env
# Edit .env with your settings

VYNILINO_ENV=development \
VYNILINO_TOKEN_KEY=$(openssl rand -hex 32) \
go run ./cmd/vynilino serve

# Run tests
go test ./...

# Regenerate GraphQL code (after schema changes)
go run github.com/99designs/gqlgen generate

# Regenerate SQL code (after query changes)
sqlc generate
```

### Project structure

```
cmd/vynilino/         # binary entrypoint and CLI commands
internal/
  adapter/
    discogs/          # Discogs API client
    filestore/        # local cover-art storage
    graphql/          # GraphQL server, middleware, resolvers
    storage/sqlite/   # SQLite repositories (sqlc-generated)
    ui/               # embedded SPA + REST helpers
  app/                # application services (auth, records, OIDC, …)
  config/             # environment-variable configuration
  ctxutil/            # request-scoped context helpers
  domain/             # core domain types (User, Record, Token, …)
openspec/             # OpenSpec specs and change archive
ui/                   # Vite + Alpine.js frontend source
web/                  # embedded assets (go:embed target)
```

## Contributing

Contributions are welcome! Please follow these steps:

1. **Open an issue** or discussion before starting significant work, so we can align on design.
2. **Fork** the repository and create a feature branch from `main`.
3. **Write tests** for any new behaviour. Run `go test ./...` before submitting.
4. **Follow the OpenSpec workflow** for non-trivial features: add a proposal under `openspec/changes/` and link it in your PR description. This keeps the decision log complete.
5. **Submit a pull request** against `main` with a clear description of what changed and why.

By contributing you agree that your contributions will be licensed under the Apache License 2.0.

## License

Copyright 2026 Thibault NORMAND

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for the full text.
