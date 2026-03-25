## 1. Project Scaffold & Configuration

- [x] 1.1 Initialize Go module (`go mod init zntr.io/vynilino`) and set up `cmd/server/main.go` entrypoint
- [x] 1.2 Define configuration struct and loader using env vars + optional YAML (`VYNILINO_DB_PATH`, `VYNILINO_MEDIA_DIR`, `VYNILINO_TOKEN_KEY`, `VYNILINO_ALLOWED_ORIGINS`, `VYNILINO_PLAYGROUND`, `VYNILINO_INTROSPECTION`, `VYNILINO_SINGLE_OWNER`)
- [x] 1.3 Add core dependencies: `go-chi/chi`, `gqlgen`, `sqlc`, `golang-migrate`, `modernc.org/sqlite`, `o1ecc8o/paseto`, `alexedwards/argon2id`
- [x] 1.4 Set up `internal/` directory structure: `domain/`, `app/`, `adapter/graphql/`, `adapter/storage/sqlite/`, `adapter/filestore/`, `config/`
- [x] 1.5 Create `Dockerfile` (multi-stage: builder + distroless/scratch runtime) and `docker-compose.yml` with volume mounts for DB and media

## 2. Database Layer

- [x] 2.1 Write initial SQL schema migration (users, records, refresh_tokens tables) in `migrations/` with `golang-migrate` naming convention
- [x] 2.2 Enable SQLite WAL mode in the connection initializer
- [x] 2.3 Write `sqlc.yaml` config targeting SQLite; define queries for users and records in `internal/adapter/storage/sqlite/queries/`
- [x] 2.4 Run `sqlc generate` to produce type-safe DB code; commit generated files
- [x] 2.5 Implement repository interfaces in `internal/domain/` (`RecordRepository`, `UserRepository`, `TokenRepository`)
- [x] 2.6 Implement SQLite adapters satisfying domain repository interfaces

## 3. User Authentication

- [x] 3.1 Implement password hashing and verification using Argon2id (memory=64MB, iterations=3, parallelism=2)
- [x] 3.2 Implement PASETO v4 local token generation and validation (access: 15min TTL, refresh: 30 day TTL)
- [x] 3.3 Implement refresh token rotation with revocation (store token hash in DB, invalidate on use)
- [x] 3.4 Implement account lockout logic (10 failures in 15 minutes → 15-minute lock)
- [x] 3.5 Implement `UserService` application service: `Register`, `Login`, `RefreshToken`, `Logout`
- [x] 3.6 Enforce single-owner mode in `Register` (reject if user count > 0)
- [x] 3.7 Write Chi authentication middleware extracting and validating Bearer token, injecting user into context

## 4. GraphQL API — Schema & Resolvers

- [x] 4.1 Write GraphQL schema (`schema.graphql`): types for `Record`, `User`, `Connection`/`Edge`/`PageInfo`; mutations `createRecord`, `updateRecord`, `deleteRecord`, `login`, `register`, `refreshToken`, `logout`; query `records`, `record`; subscription `recordChanged`
- [x] 4.2 Run `gqlgen generate` to produce resolver stubs and models
- [x] 4.3 Implement `RecordResolver` (createRecord, updateRecord, deleteRecord, record, records) wiring to `RecordService`
- [x] 4.4 Implement `AuthResolver` (login, register, refreshToken, logout) wiring to `UserService`
- [x] 4.5 Implement cursor-based pagination helper for `records` query
- [x] 4.6 Implement search and filter logic in `RecordService.List` (full-text, field filters, sort)
- [x] 4.7 Implement duplicate detection warning in `createRecord`
- [x] 4.8 Add query depth limit (max 10) and complexity limit (budget 1000) middleware in gqlgen config
- [x] 4.9 Add request body size limit middleware (1MB) on the Chi router

## 5. GraphQL Subscriptions

- [x] 5.1 Add WebSocket handler using `graphql-transport-ws` protocol via gqlgen subscription transport
- [x] 5.2 Implement `recordChanged` subscription resolver publishing events on record mutations
- [x] 5.3 Implement token validation in WebSocket connection init payload

## 6. Media Management

- [x] 6.1 Implement `POST /media/cover-art` HTTP handler: validate MIME type by magic bytes (JPEG/PNG/WebP), enforce 5MB limit, store file under `VYNILINO_MEDIA_DIR/<userId>/<recordId>.<ext>`, associate URL with record
- [x] 6.2 Implement `GET /media/cover-art/:id` HTTP handler: authenticate request, serve file with `Content-Type` and `Cache-Control: private, max-age=86400`
- [x] 6.3 Implement cover art deletion in `deleteRecord` (remove file from disk after DB deletion)
- [x] 6.4 Implement `FileStore` interface and local filesystem adapter

## 7. Data Portability

- [x] 7.1 Implement `GET /export/json` handler: stream all user records as JSON with `Content-Disposition` attachment header
- [x] 7.2 Implement `GET /export/csv` handler: stream RFC 4180 CSV with proper quoting
- [x] 7.3 Implement `POST /import/csv` handler: parse CSV, map columns to record fields, insert valid rows, return import summary
- [x] 7.4 Implement Discogs CSV column detection and mapping (detect by presence of "Catalog#" or "release_id" headers)
- [x] 7.5 Enforce 10MB upload limit on import endpoint

## 8. CORS & Security Headers

- [x] 8.1 Add Chi CORS middleware reading `VYNILINO_ALLOWED_ORIGINS`
- [x] 8.2 Add security headers middleware: `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`, `Content-Security-Policy` for API responses
- [x] 8.3 Conditionally serve GraphQL Playground at `GET /graphql` based on `VYNILINO_PLAYGROUND` flag
- [x] 8.4 Conditionally enable introspection based on `VYNILINO_INTROSPECTION` flag

## 9. Testing

- [x] 9.1 Write unit tests for `UserService` (registration, login, lockout, token refresh) using in-memory repository stubs
- [x] 9.2 Write integration tests for SQLite adapters using a temporary file DB (no mocks)
- [x] 9.3 Write integration tests for GraphQL resolvers using `httptest` and real SQLite DB
- [x] 9.4 Write tests for CSV import/export correctness including special characters and Discogs mapping
- [x] 9.5 Write tests for media upload validation (MIME type, size limits)
- [x] 9.6 Write tests for auth middleware (missing token, expired token, valid token)

## 10. Operational Readiness

- [x] 10.1 Add structured logging using `log/slog` (JSON format in production, text in development)
- [x] 10.2 Add health check endpoint `GET /health` returning `{"status":"ok","db":"ok"}` (checks DB connectivity)
- [x] 10.3 Add graceful shutdown (SIGTERM/SIGINT): drain in-flight requests, close DB connection
- [x] 10.4 Add `--check-migrations` CLI flag that runs migration dry-run and exits 0/1
- [x] 10.5 Write `README.md` with self-hosting quickstart: env vars reference, Docker Compose example, backup instructions, Discogs import guide
