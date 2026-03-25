## Context

Vynilino's backend currently uses Go's `flag` package with a single `cmd/server/main.go` that wires the entire application through `go.uber.org/fx`. The frontend is a Vite/Alpine.js SPA built as a single entry point, embedding all views (including Login) into one JS bundle served from `web/dist/` via an embedded FS in the Go binary.

Two independent concerns are addressed here: (1) reduce the JS payload for unauthenticated users by splitting login into its own bundle, and (2) replace the `flag`-based backend entrypoint with a Cobra CLI so that `vynilino serve` starts the daemon and `vynilino user *` handles admin operations without requiring database access tooling.

## Goals / Non-Goals

**Goals:**
- Serve a minimal login bundle (`~login entry`) to unauthenticated users; full app bundle only loads after authentication.
- Replace `cmd/server/main.go` with a Cobra root command at `cmd/vynilino/main.go`.
- `vynilino serve` is the new way to start the HTTP server (same behavior, same env vars).
- `vynilino user add|deactivate|activate|change-password|list` operates on the SQLite DB via the existing domain layer.
- Admin CLI commands run without starting the HTTP server.

**Non-Goals:**
- No REST/GraphQL API changes for user management (admin CLI is out-of-band).
- No new authentication mechanism — admin commands rely on local file-system access to the DB (server-side only, not remotely exposed).
- No multi-user role system changes.
- No frontend framework swap or new UI library.

## Decisions

### 1. Cobra for CLI framework

**Decision**: Use `github.com/spf13/cobra` with a root command and two subcommand groups: `serve` and `user`.

**Rationale**: Cobra is the de-facto standard in the Go ecosystem, provides `--help` generation, flag inheritance, and shell completion out of the box. The existing `flag` usage is minimal and migrates trivially.

**Alternative considered**: `urfave/cli` — similar capability but less idiomatic for Go server tooling. Rejected to stay consistent with broader ecosystem conventions.

### 2. Two Vite entry points (login vs app)

**Decision**: Add `src/login.html` + `src/login.js` as a second Vite entry point. `src/index.html` + `src/main.js` remain the authenticated app shell.

**Rationale**: Vite natively supports multi-page apps (MPA) via the `build.rollupOptions.input` map. This produces two completely independent HTML+JS bundles with no shared chunk between them, so the login page loads zero application code. The backend serves `login.html` for `/login` and `index.html` for all other SPA paths.

**Alternative considered**: Dynamic `import()` code-splitting within a single entry — this reduces bundle parse time on first load but still downloads shared runtime chunks. Full separation is simpler and more robust for this scale.

### 3. Admin CLI reuses existing domain/repository layer

**Decision**: Admin commands instantiate repositories directly (open SQLite, run migrations if needed) without starting fx or the HTTP server.

**Rationale**: The `domain.UserRepository` and `app.UserService` interfaces are already well-defined. Wiring them directly in command handlers avoids pulling in the full fx graph for simple CRUD operations. This keeps admin commands fast and dependency-light.

**Alternative considered**: Running the full fx app with an "admin mode" flag — rejected because it would start HTTP listeners unnecessarily and couples concerns.

### 4. Binary rename: `server` → `vynilino`

**Decision**: Move entrypoint from `cmd/server/` to `cmd/vynilino/`.

**Rationale**: With a CLI, the binary should be named after the product, not the mode it runs in. `vynilino serve` is more discoverable than `server` with no subcommands. Makefile and Dockerfile are updated accordingly.

### 5. Serve login.html for `/login` from embedded FS

**Decision**: The HTTP handler in `internal/adapter/ui/handler.go` is extended to serve `dist/login.html` when the request path is `/login` (or `/login.html`). All other non-API, non-asset paths continue to serve `dist/index.html`.

**Rationale**: No new server-side routing complexity — it's a single extra `if` branch. The embedded FS already contains both files after the Vite build.

## Risks / Trade-offs

- **Cache invalidation for login bundle**: Since `login.js` is a separate hashed asset, browsers that cached the old single-bundle URL will fetch the new login bundle on next visit automatically. No special migration needed.
- **Admin CLI data consistency**: Admin commands open the SQLite file directly while the server may be running. SQLite's WAL mode (already in use) handles concurrent readers, but schema-changing operations (none planned here) would need coordination. → Mitigation: Admin commands are read/write at row level only; no DDL operations.
- **fx removal from admin path**: Admin commands bypass fx lifecycle hooks (OnStart/OnStop). This is intentional but means any future cross-cutting concerns added to fx hooks won't automatically apply to admin. → Mitigation: Document that admin commands are not lifecycle-managed.
- **Cobra version**: `cobra` v1.x is stable and widely adopted. No version risk.

## Migration Plan

1. Add `github.com/spf13/cobra` to `go.mod` (`go get`).
2. Create `cmd/vynilino/main.go` with root, `serve`, and `user` commands.
3. Move all fx wiring from `cmd/server/main.go` into `cmd/vynilino/cmd/serve.go`.
4. Update `Makefile`: change `go build ./cmd/server` → `go build ./cmd/vynilino`.
5. Update `Dockerfile`: update `COPY` and `CMD` to use new binary path.
6. Add `src/login.html` and `src/login.js` to the Vite project.
7. Update `vite.config.js` to declare both entry points.
8. Update `internal/adapter/ui/handler.go` to route `/login` to `login.html`.
9. Delete `cmd/server/` directory.

**Rollback**: The old `cmd/server/` can be restored from git. No database changes means no rollback complexity.

## Open Questions

- Should `vynilino user add` prompt for password interactively (masked input) or accept `--password` flag? Interactive is safer for shell history. → Prefer interactive with `--password-stdin` flag as alternative for scripted use.
- Should `vynilino serve` support a `--check-migrations` flag (migrating the existing `-check-migrations` flag)? → Yes, preserve as `vynilino serve --check-migrations`.
