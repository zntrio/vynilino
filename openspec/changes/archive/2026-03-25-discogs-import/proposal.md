## Why

Users currently must manually enter all vinyl metadata (artist, title, label, year, tracklist, etc.) when adding records to their collection — a tedious and error-prone process. Discogs maintains the world's largest crowd-sourced music database, and integrating with their search API allows users to find and import accurate vinyl metadata in seconds rather than minutes.

## What Changes

- Add a Discogs search endpoint/flow accessible from the "Add Record" UI
- Allow users to search Discogs by artist, album title, or barcode
- Display search results from the Discogs API with cover art and key metadata
- Let users select a Discogs release and auto-populate the record creation form
- Store the `discogs_id` on the record for future reference/sync

## Capabilities

### New Capabilities

- `discogs-integration`: Search the Discogs database and import release metadata (title, artist, label, year, format, tracklist, cover image) to pre-fill a new vinyl collection entry.

### Modified Capabilities

- `collection-management`: Record creation flow gains an optional "Import from Discogs" path; the Record domain model gains a `discogs_id` field.
- `collection-ui`: The Add Record view gains a Discogs search panel before or alongside the manual entry form.

## Impact

- **New external dependency**: Discogs REST API (`api.discogs.com`). Requires a User-Agent header; optionally a Discogs API token for higher rate limits.
- **Backend**: New service/use-case for calling Discogs search + release endpoints; new GraphQL query or mutation to proxy the search to the frontend.
- **Database**: `discogs_id` column added to the `records` table (nullable, for records imported from Discogs).
- **Frontend**: New search UI component in the Add Record view.
- **No breaking changes** to existing collection APIs or data.
