## Context

Vynilino is a Go + GraphQL backend with a Vite/Vanilla JS frontend. Records are stored in SQLite via sqlc-generated queries. The domain model (`Record`) has no external-source reference field today. The Discogs API is a public REST API (`https://api.discogs.com`) requiring only a `User-Agent` header; an optional personal access token raises the rate limit from 25 req/min (unauthenticated) to 60 req/min.

The import flow must fit the existing "Add Record" UX without breaking the current manual creation path.

## Goals / Non-Goals

**Goals:**
- Search Discogs releases by artist/title/barcode from the frontend
- Display a list of matching Discogs results with key metadata (artist, title, year, label, format, cover thumbnail)
- Allow selecting a result to auto-populate the `CreateRecordInput` form
- Store `discogs_id` on the record for traceability
- Keep the manual creation path fully intact

**Non-Goals:**
- Full Discogs sync / collection mirroring
- OAuth login via Discogs
- Importing the user's existing Discogs collection in bulk
- Writing back to Discogs (e.g. marking as "for sale")
- Tracklist or full release detail import in v1

## Decisions

### 1. Proxy Discogs via backend, not direct frontend calls

**Decision**: The frontend calls a new GraphQL query `searchDiscogs(query, type)` which proxies to the Discogs API server-side.

**Rationale**: Keeps the Discogs token out of the browser, avoids CORS issues, and allows adding caching or rate-limit management later without frontend changes.

**Alternative considered**: Direct frontend fetch to `api.discogs.com`. Rejected because Discogs CORS headers do not allow arbitrary origins in production and the token would be exposed.

---

### 2. New `DiscogsResult` GraphQL type + `searchDiscogs` query

**Decision**: Add a `searchDiscogs(query: String!, type: DiscogsSearchType): [DiscogsResult!]!` query returning a lightweight `DiscogsResult` type.

```graphql
enum DiscogsSearchType { RELEASE MASTER ALL }

type DiscogsResult {
  discogsId: String!
  title: String!
  artist: String
  year: Int
  label: String
  format: String
  thumbUrl: String
  country: String
}
```

**Rationale**: Mirrors the Discogs `/database/search` response fields we care about. Reusing `Record` would create confusion since a `DiscogsResult` is not yet a persisted record.

---

### 3. `discogs_id` nullable column on `records` table

**Decision**: Add `discogs_id TEXT` (nullable) to the `records` table via a new migration `000004_discogs_id`. Expose it on the `Record` GraphQL type as `discogsId: String`.

**Rationale**: Provides traceability and enables future de-duplication ("you already have this release"). Nullable because most existing records will have no Discogs link.

**Alternative considered**: Store in a separate `record_sources` table. Rejected as over-engineering for a single optional field.

---

### 4. `CreateRecordInput` gains optional `discogsId`

**Decision**: Add `discogsId: String` to `CreateRecordInput` so the frontend can pass the Discogs release ID when creating from an import.

**Rationale**: Ties the created record back to the Discogs source atomically at creation time, no separate mutation needed.

---

### 5. Cover art: use Discogs `thumb` URL or download

**Decision**: For v1, store the Discogs thumbnail URL directly as `coverArtUrl` on the record (no download/re-upload to local filestore).

**Rationale**: Avoids complexity of downloading and storing cover art during import. Discogs CDN URLs are stable. Can be upgraded to local storage in a later change.

**Risk**: Discogs could change URL structure. Acceptable for v1.

---

### 6. Backend Discogs client as internal adapter

**Decision**: Implement a `DiscogsClient` in `internal/adapter/discogs/client.go` with a simple `Search(ctx, query, searchType string) ([]DiscogsResult, error)` method. Wire it into the GraphQL resolver. Use `https://github.com/irlndts/go-discogs` library.

**Rationale**: Keeps the HTTP client isolated, easy to mock in tests, and consistent with existing adapter pattern (filestore, storage/sqlite).

## Risks / Trade-offs

- **Rate limiting** → Discogs caps unauthenticated requests at 25/min. Mitigation: read `VYNILINO_DISCOGS_TOKEN` from env for authenticated requests (60/min). Add 429 handling with a descriptive GraphQL error.
- **Discogs availability** → If Discogs is down, search fails gracefully with an error; manual creation is unaffected. Mitigation: set reasonable HTTP timeout (5s).
- **Cover art hotlinking** → Discogs may rate-limit image requests for records imported this way. Mitigation: acceptable in v1; future change can download art.
- **Data quality** → Discogs search results can be noisy (multiple versions of same release). Mitigation: return top results and let user choose; display format/country/year to help distinguish.

## Migration Plan

1. Add migration `000004_discogs_id` (non-breaking, nullable column)
2. Re-run `sqlc generate` to update generated query code
3. Deploy backend — existing records unaffected (`discogs_id` defaults to NULL)
4. Deploy frontend — Discogs search panel is additive; manual path unchanged
5. Rollback: drop migration, revert code; no data loss (column is nullable and only set on new imports)
