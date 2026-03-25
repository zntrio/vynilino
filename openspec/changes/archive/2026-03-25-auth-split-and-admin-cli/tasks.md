## 1. Dependencies & Repo Structure

- [x] 1.1 Add `github.com/spf13/cobra` to `go.mod` and `go.sum` (`go get github.com/spf13/cobra`)
- [x] 1.2 Create directory `cmd/vynilino/` with subdirectory `cmd/vynilino/internal/cmd/`
- [x] 1.3 Delete `cmd/server/` directory after the new entrypoint is verified

## 2. Backend CLI — Root & Serve Command

- [x] 2.1 Create `cmd/vynilino/main.go` with a Cobra root command (`vynilino`) that prints help when called with no subcommand
- [x] 2.2 Create `cmd/vynilino/internal/cmd/serve.go` with the `serve` subcommand; migrate all fx wiring from the old `cmd/server/main.go` into this file
- [x] 2.3 Migrate the `--check-migrations` flag to `vynilino serve --check-migrations`
- [x] 2.4 Verify `vynilino serve` starts the HTTP server correctly with all existing env vars

## 3. Backend CLI — User Admin Commands

- [x] 3.1 Create `cmd/vynilino/internal/cmd/user.go` with the `user` parent command group
- [x] 3.2 Implement `vynilino user list` — open DB, query all users, print tabular output (ID, email, status, created-at)
- [x] 3.3 Implement `vynilino user add --email <email>` with interactive password prompt (masked) and `--password-stdin` flag; reuse `app.UserService` for creation
- [x] 3.4 Implement `vynilino user deactivate --email <email>` — set user status to inactive
- [x] 3.5 Implement `vynilino user activate --email <email>` — re-enable an inactive user
- [x] 3.6 Implement `vynilino user change-password --email <email>` with interactive prompt and `--password-stdin` flag
- [x] 3.7 Add `--db` flag (default from `VYNILINO_DB_PATH` env var, fallback `./vynilino.db`) to all `user` subcommands via persistent flag on the `user` command

## 4. User Domain — Status Field

- [x] 4.1 Add `Active bool` (or `Status` field) to `domain.User` if not already present
- [x] 4.2 Add `DeactivateUser(ctx, email)` and `ActivateUser(ctx, email)` methods to `domain.UserRepository` interface
- [x] 4.3 Implement `DeactivateUser` and `ActivateUser` in `internal/adapter/storage/sqlite/user_repo.go`
- [x] 4.4 Add corresponding SQL queries to `internal/adapter/storage/sqlite/queries/users.sql` and regenerate sqlc
- [x] 4.5 Write a DB migration (`000003_user_active.up.sql` / `.down.sql`) that adds an `active` column (default `TRUE`) to the `users` table
- [x] 4.6 Enforce `active` check in `app.UserService.Login` — inactive users SHALL receive `ACCOUNT_LOCKED` or a new `ACCOUNT_DISABLED` error

## 5. Makefile & Dockerfile Updates

- [x] 5.1 Update `Makefile` build target: change `go build ./cmd/server` → `go build -o vynilino ./cmd/vynilino`
- [x] 5.2 Update `Dockerfile`: change `COPY` source path and `CMD`/`ENTRYPOINT` to use `vynilino serve`
- [x] 5.3 Verify `make build` produces a working binary named `vynilino`

## 6. Frontend — Login Bundle Split

- [x] 6.1 Create `ui/src/login.html` — minimal HTML shell that loads `login.js` (no app-shell markup)
- [x] 6.2 Create `ui/src/login.js` — self-contained Alpine.js entry: initialise Alpine, define auth + toast stores, render login form component
- [x] 6.3 Extract the login form logic from `ui/src/views/Login.js` into the standalone `login.js` entry (keep `Login.js` for use within the app's router for `/login` deep-link fallback if needed, or remove it)
- [x] 6.4 Update `ui/vite.config.js`: add `build.rollupOptions.input` with both `index: 'src/index.html'` and `login: 'src/login.html'`
- [ ] 6.5 Verify the Vite build produces separate `dist/index.html` and `dist/login.html` with no shared JS chunk between them

## 7. Backend — Static File Routing Update

- [x] 7.1 Update `internal/adapter/ui/handler.go`: add a case to serve `dist/login.html` when the request path is `/login` or `/login.html`
- [x] 7.2 Ensure the SPA fallback (`index.html`) still applies for all other non-API, non-asset paths excluding `/login`
- [x] 7.3 Update handler tests in `internal/adapter/ui/handler_test.go` to cover the `/login` → `login.html` route

## 8. Integration & Verification

- [x] 8.1 Run `go test ./...` and confirm all existing tests pass
- [ ] 8.2 Run `vynilino serve` and confirm the app starts; access `/login` → verify login.html is served
- [ ] 8.3 Run `vynilino user list`, `user add`, `user deactivate`, `user activate`, `user change-password` against a dev DB and confirm correct behaviour
- [ ] 8.4 Build and run the Docker image (`docker compose up`) and verify `vynilino serve` starts correctly
