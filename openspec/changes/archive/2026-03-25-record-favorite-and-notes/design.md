## Context

Records currently store factual metadata (title, artist, year, label, condition, etc.). There is no way for a user to mark a record as special or attach a private thought to it. Both features are user-scoped and have no shared or exported meaning — they are purely personal annotations on top of the catalog data.

## Goals / Non-Goals

**Goals:**
- Add a `favorite` boolean to each record, togglable per-user
- Add a `personalNote` free-text field to each record, private to the owning user
- Extend the GraphQL API and UI to expose both fields
- Allow filtering the collection by `favorite = true`

**Non-Goals:**
- Shared or collaborative notes (notes remain strictly per-user)
- Exporting personal notes in data-portability flows (out of scope for now)
- Rich-text or markdown formatting for notes (plain text only)
- "Favorites" as a separate shareable list or public profile feature

## Decisions

### 1. Store on the `records` table, not a separate table

**Decision**: Add `favorite` and `personal_note` columns directly to `records`.

**Rationale**: Records are already user-scoped (`user_id` FK). A separate table would add a join for every record fetch with no benefit — there is a 1:1 relationship between a record and its personal annotation. Denormalising here is appropriate.

**Alternative considered**: Separate `record_annotations` table. Rejected: extra join, extra sqlc boilerplate, no multi-user scenario to justify it.

### 2. Extend `updateRecord` mutation rather than adding dedicated mutations

**Decision**: Add `favorite` and `personalNote` to the existing `UpdateRecordInput` rather than introducing `setFavorite` / `setPersonalNote` mutations.

**Rationale**: Keeps the API surface minimal. Clients already know how to call `updateRecord`; adding fields there requires zero new resolver logic. A dedicated mutation would only be justified if the fields needed different auth or rate-limiting rules — they don't.

**Alternative considered**: Separate `toggleFavorite(id)` mutation for UX convenience. Rejected: the UI can trivially read the current value and send its negation via `updateRecord`.

### 3. Filter favorites via existing `records` query filter argument

**Decision**: Extend the `RecordFilter` input with an optional `favoritesOnly: Boolean` flag.

**Rationale**: Consistent with existing search/filter patterns (genre, format). No new query endpoint needed.

## Risks / Trade-offs

- [Personal notes are plain text] → No XSS risk server-side (stored as-is), but UI must HTML-escape on render. Standard escaping already applied.
- [Migration adds two columns to `records`] → SQLite `ALTER TABLE ADD COLUMN` is safe and non-blocking. Default values (`0` / `NULL`) mean no backfill needed.
- [Personal notes excluded from export] → Users may expect their notes to be included in a data export. Mitigated by leaving a clear TODO in the data-portability spec for a future change.

## Migration Plan

1. Add migration `000005_record_personal_data.up.sql`: `ALTER TABLE records ADD COLUMN favorite INTEGER NOT NULL DEFAULT 0; ALTER TABLE records ADD COLUMN personal_note TEXT;`
2. Add corresponding `.down.sql` to drop the columns (SQLite requires table recreation — use standard pattern).
3. Update sqlc query files and regenerate.
4. No data backfill required.
5. Rollback: run `.down.sql`; existing data unaffected.
