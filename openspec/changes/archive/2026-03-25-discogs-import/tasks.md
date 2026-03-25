## 1. Database Migration

- [x] 1.1 Create migration `000004_discogs_id.up.sql` adding nullable `discogs_id TEXT` column to the `records` table
- [x] 1.2 Create migration `000004_discogs_id.down.sql` dropping the `discogs_id` column
- [x] 1.3 Update `sqlc.yaml` and `queries/records.sql` to include `discogs_id` in all record SELECT/INSERT queries
- [x] 1.4 Run `sqlc generate` and verify the generated `sqlcdb/records.sql.go` includes the new field

## 2. Domain Model

- [x] 2.1 Add `DiscogsID *string` field to `internal/domain/record.go` `Record` struct
- [x] 2.2 Add `DiscogsID *string` field to `CreateRecordInput` equivalent in the domain layer (if any)

## 3. Discogs HTTP Client

- [x] 3.1 Create `internal/adapter/discogs/client.go` with a `Client` struct that holds HTTP client, base URL, optional token, and User-Agent
- [x] 3.2 Implement `Client.Search(ctx context.Context, query, searchType string) ([]SearchResult, error)` calling `GET https://api.discogs.com/database/search`
- [x] 3.3 Define `SearchResult` struct mapping Discogs API response fields: `ID`, `Title`, `Artist`, `Year`, `Label`, `Format`, `Thumb`, `Country`
- [x] 3.4 Read `VYNILINO_DISCOGS_TOKEN` in the client constructor; set `Authorization: Discogs token=<value>` header when present
- [x] 3.5 Set a 5-second HTTP timeout and a descriptive `User-Agent` header (e.g. `vynilino/1.0`)
- [x] 3.6 Handle HTTP 429 by returning a typed `ErrRateLimit` error; handle non-2xx responses with a wrapped error

## 4. Application Layer

- [x] 4.1 Add `SearchDiscogs(ctx context.Context, query, searchType string) ([]domain.DiscogsResult, error)` use-case in `internal/app/record.go` (or new `discogs.go`)
- [x] 4.2 Define `domain.DiscogsResult` struct mirroring the GraphQL `DiscogsResult` type
- [x] 4.3 Wire the `discogs.Client` into the app service (pass as interface to allow mocking)
- [x] 4.4 Propagate `ErrRateLimit` and timeout errors as user-friendly messages to the GraphQL layer

## 5. Storage Layer

- [x] 5.1 Update `internal/adapter/storage/sqlite/record_repo.go` `Create` method to persist `discogs_id` from the domain `Record`
- [x] 5.2 Update `record_repo.go` scan logic so `GetByID`, `List`, and `Update` return the `discogs_id` field correctly

## 6. GraphQL Schema & Resolvers

- [x] 6.1 Add `DiscogsSearchType` enum (`RELEASE`, `MASTER`, `ALL`) to `schema.graphql`
- [x] 6.2 Add `DiscogsResult` type to `schema.graphql` with fields: `discogsId`, `title`, `artist`, `year`, `label`, `format`, `thumbUrl`, `country`
- [x] 6.3 Add `searchDiscogs(query: String!, type: DiscogsSearchType): [DiscogsResult!]!` query to `schema.graphql`
- [x] 6.4 Add `discogsId: String` field to the existing `Record` type in `schema.graphql`
- [x] 6.5 Add `discogsId: String` field to `CreateRecordInput` in `schema.graphql`
- [x] 6.6 Run `go generate` (gqlgen) to regenerate `graph/generated.go` and `graph/models_gen.go`
- [x] 6.7 Implement `searchDiscogs` resolver in `schema.resolvers.go` — call the app layer use-case, map results, handle errors with GraphQL error extensions
- [x] 6.8 Update the `createRecord` resolver to pass `discogsId` from the input to the domain `CreateRecord` call
- [x] 6.9 Ensure `searchDiscogs` resolver enforces authentication (returns auth error for unauthenticated callers)

## 7. Configuration

- [x] 7.1 Add `DiscogsToken string` field to `internal/config/config.go` read from `VYNILINO_DISCOGS_TOKEN` env var
- [x] 7.2 Pass the token from config to the `discogs.Client` constructor in the server wiring

## 8. Frontend

- [x] 8.1 Add a `searchDiscogs` GraphQL query helper in `ui/src/lib/gql.js`
- [x] 8.2 Create `ui/src/components/DiscogsSearch.js` Alpine.js component with search input, loading state, results list, and error display
- [x] 8.3 Integrate `DiscogsSearch` into `ui/src/views/AddRecord.js` as a toggle panel above the manual form
- [x] 8.4 On result selection, pre-populate form fields: title, artist, year, label, format, coverArtUrl (thumb), discogsId
- [x] 8.5 Display cover thumbnail in search results list
- [x] 8.6 Show "No results found" state and error state messages per spec

## 9. Tests

- [x] 9.1 Unit test `discogs.Client.Search` with a mock HTTP server covering: success, 429, timeout, non-2xx
- [x] 9.2 Unit test the `SearchDiscogs` app use-case with a mock Discogs client
- [x] 9.3 Update SQLite integration test to verify `discogs_id` is persisted and returned correctly
- [x] 9.4 Add GraphQL resolver test for `searchDiscogs` (authenticated and unauthenticated cases)
