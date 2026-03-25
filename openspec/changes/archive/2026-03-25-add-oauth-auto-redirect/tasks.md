## 1. Configuration

- [x] 1.1 Add `OIDCAutoRedirect bool` field to `Config` struct in `internal/config/config.go`
- [x] 1.2 Load `VYNILINO_OIDC_AUTO_REDIRECT` via `getBoolEnv` (default `false`) in `config.Load()`

## 2. Login Redirect Handler

- [x] 2.1 Implement `loginRedirectHandler(cfg *config.Config, oidcSvc *app.OIDCService, staticFS fs.FS) http.HandlerFunc` in `internal/adapter/graphql/` (or a thin adjacent file)
- [x] 2.2 When `cfg.OIDCAutoRedirect && oidcSvc != nil`: call `oidcSvc.AuthorizationURL(ctx)`, respond with HTTP 302 and `Location` header
- [x] 2.3 When `AuthorizationURL` returns an error: respond with HTTP 500 and a minimal HTML error page ("OIDC provider unavailable. Contact your administrator.")
- [x] 2.4 Otherwise (flag off or OIDC disabled): serve `login.html` from `staticFS` (same logic as current `SPAHandler` `/login` branch)
- [x] 2.5 Register `r.Get("/login", loginRedirectHandler(cfg, oidcSvc, staticFiles))` in `server.go` before `r.Mount("/", uiHandler.SPAHandler())`
- [x] 2.6 Remove the `/login` special-case from `ui.Handler.SPAHandler()` since it is now handled by the explicit route

## 3. Tests

- [x] 3.1 Unit test: flag off → handler serves `login.html` (200)
- [x] 3.2 Unit test: flag on, `oidcSvc` nil → handler serves `login.html` (200)
- [x] 3.3 Unit test: flag on, `oidcSvc` returns URL → handler responds 302 with correct `Location`
- [x] 3.4 Unit test: flag on, `oidcSvc` returns error → handler responds 500 with error HTML body

## 4. Verification

- [x] 4.1 Run `go test ./internal/config/... ./internal/adapter/graphql/...` — all tests pass
- [ ] 4.2 Manual smoke test: start server with `VYNILINO_OIDC_AUTO_REDIRECT=true` and valid OIDC env vars; `curl -I http://localhost:8080/login` returns `302` with `Location` pointing to the OIDC provider
- [ ] 4.3 Manual smoke test: start server without `VYNILINO_OIDC_AUTO_REDIRECT`; `GET /login` returns the login page HTML unchanged
