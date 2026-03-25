## Why

The current frontend ships a single JavaScript bundle that includes login UI, application views, and all dependencies — this means unauthenticated users download the entire app just to see a login form. Separately, the backend has no administrative interface: managing users (adding, deactivating, resetting passwords) requires direct database access. Both gaps increase maintenance burden and operational friction.

## What Changes

- **Frontend login split**: The Vite build is restructured into two separate entry points — `login.html` / `login.js` for unauthenticated users, and `index.html` / `main.js` for the authenticated app. The server routes unauthenticated requests to the login shell; authenticated sessions load the full app.
- **Backend CLI refactor**: `cmd/server/main.go` is replaced by a Cobra-based CLI. The HTTP server starts with `vynilino serve`. All existing flag-based logic migrates to `serve` subcommand flags/env vars.
- **Admin subcommands**: New `vynilino user` command group exposes: `user add`, `user deactivate`, `user activate`, `user change-password`, `user list`. All commands operate directly against the SQLite database via the existing domain layer.
- The binary name stays `vynilino`; the `serve` subcommand becomes the canonical way to start the daemon.

## Capabilities

### New Capabilities

- `admin-cli`: Embedded Cobra CLI for backend administrative operations (user management, server startup). Replaces ad-hoc `flag` usage and adds `user` subcommand group.

### Modified Capabilities

- `collection-ui`: Login page is served from a separate entry point (`/login.html`) and bundle so unauthenticated users do not download the full application JavaScript.

## Impact

- **`cmd/server/main.go`**: Replaced by `cmd/vynilino/main.go` with Cobra root command, `serve` subcommand, and `user` subcommand group.
- **`ui/`**: `vite.config.js` gains a second entry point; `src/login.html` and `src/login.js` are new files; `src/main.js` and `src/index.html` remain (app shell, authenticated only).
- **`internal/adapter/ui/handler.go`**: Static file routing must serve `login.html` for `/login` path and `index.html` for all authenticated paths.
- **`web/embed.go`**: Embedded FS still works; no change to embed directive, but `dist/` will now contain both `index.html` and `login.html`.
- **Dependencies**: `github.com/spf13/cobra` added to `go.mod`.
- **`Makefile` / `Dockerfile`**: Binary path changes from `cmd/server` to `cmd/vynilino`.
- No GraphQL schema changes; no database migrations required.
