## 1. Dependencies and Configuration

- [x] 1.1 Add `github.com/coreos/go-oidc/v3` and `golang.org/x/oauth2` to `go.mod` / `go.sum`
- [x] 1.2 Add four OIDC fields to `internal/config/config.go`: `OIDCIssuer`, `OIDCClientID`, `OIDCClientSecret`, `OIDCRedirectURL` (read from `VYNILINO_OIDC_*` env vars)
- [x] 1.3 Add OIDC fields to the env-vars table in `README.md`

## 2. Domain

- [x] 2.1 Add `OIDCIdentity` entity to `internal/domain/user.go` (fields: `UserID`, `Provider`, `Subject`, `CreatedAt`)
- [x] 2.2 Add `OIDCIdentityRepository` interface to `internal/domain/user.go` with methods: `FindByProviderSubject(ctx, provider, subject) (*OIDCIdentity, error)` and `Create(ctx, *OIDCIdentity) error`
- [x] 2.3 Add `OIDCState` entity to `internal/domain/oidc.go` (fields: `State`, `Nonce`, `CodeVerifier`, `CreatedAt`) and `OIDCStateRepository` interface with `Create`, `FindByState`, `Delete` methods

## 3. Database Migration

- [x] 3.1 Create `internal/adapter/storage/sqlite/migrations/000002_oidc.up.sql` with `oidc_identities` table (`user_id`, `provider`, `subject`, `created_at`, unique on `(provider, subject)`) and `oidc_states` table (`state`, `nonce`, `code_verifier`, `created_at`)
- [x] 3.2 Create `internal/adapter/storage/sqlite/migrations/000002_oidc.down.sql` dropping both tables

## 4. sqlc Queries

- [x] 4.1 Create `internal/adapter/storage/sqlite/queries/oidc_identities.sql` with queries: `CreateOIDCIdentity`, `FindOIDCIdentityByProviderSubject`
- [x] 4.2 Create `internal/adapter/storage/sqlite/queries/oidc_states.sql` with queries: `CreateOIDCState`, `FindOIDCStateByState`, `DeleteOIDCState`, `DeleteExpiredOIDCStates`
- [x] 4.3 Run `sqlc generate` and commit generated files under `internal/adapter/storage/sqlite/sqlcdb/`

## 5. Repository Implementations

- [x] 5.1 Create `internal/adapter/storage/sqlite/oidc_identity_repo.go` implementing `domain.OIDCIdentityRepository` using sqlc-generated queries
- [x] 5.2 Create `internal/adapter/storage/sqlite/oidc_state_repo.go` implementing `domain.OIDCStateRepository` using sqlc-generated queries

## 6. Application Service

- [x] 6.1 Create `internal/app/oidc.go` with `OIDCService` struct holding `cfg *config.Config`, `userSvc *UserService`, `identityRepo domain.OIDCIdentityRepository`, `stateRepo domain.OIDCStateRepository`, and a lazily-initialised `*oidc.Provider`
- [x] 6.2 Implement `OIDCService.AuthorizationURL(ctx) (url, state string, err error)`: generate random `state`/`nonce`/`code_verifier`, build PKCE challenge (S256), store state record, return full authorization URL
- [x] 6.3 Implement `OIDCService.HandleCallback(ctx, code, state string) (*AuthPayload, error)`: look up state record, exchange code, verify ID token (nonce, sig, iss, aud, exp), resolve user via sub/email claims (auto-provision or link), issue PASETO tokens via `userSvc`, delete state record
- [x] 6.4 Add `NewOIDCService` constructor; return `nil, nil` when `cfg.OIDCIssuer == ""` so fx can treat it as optional

## 7. GraphQL Schema and Resolvers

- [x] 7.1 Add `OIDCAuthURLPayload` type and `oidcAuthorizationURL` / `oidcCallback` mutations to `internal/adapter/graphql/graph/schema.graphql`
- [x] 7.2 Run `go run github.com/99designs/gqlgen generate` to regenerate resolver stubs
- [x] 7.3 Implement `oidcAuthorizationURL` resolver in `schema.resolvers.go`: guard on `OIDCService == nil`, delegate to `oidcSvc.AuthorizationURL`
- [x] 7.4 Implement `oidcCallback` resolver in `schema.resolvers.go`: guard on `OIDCService == nil`, delegate to `oidcSvc.HandleCallback`
- [x] 7.5 Add `OIDCSvc *app.OIDCService` field to `graph.Resolver` and thread it through `NewRouter` / `newGraphQLHandler`

## 8. Dependency Injection

- [x] 8.1 Register `sqlite.NewOIDCIdentityRepository`, `sqlite.NewOIDCStateRepository`, and `app.NewOIDCService` in the `fx.New(...)` call in `cmd/server/main.go`

## 9. Tests

- [x] 9.1 Add `internal/adapter/storage/sqlite/integration_test.go` cases for `OIDCIdentityRepository` (`Create`, `FindByProviderSubject`) and `OIDCStateRepository` (`Create`, `FindByState`, `DeleteExpiredOIDCStates`)
- [x] 9.2 Create `internal/app/oidc_test.go` with unit tests using in-memory stubs: OIDC disabled when issuer empty, state expiry, invalid state, unverified email skips linking, auto-provision blocked in single-owner mode, successful callback issues tokens
- [x] 9.3 Verify `go test ./...` passes
