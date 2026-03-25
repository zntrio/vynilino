## Why

Users want to curate their collection beyond catalog data — marking records as favorites surfaces personal highlights, and attaching a private note captures personal memories or context that is entirely separate from factual record metadata.

## What Changes

- Add a `favorite` boolean flag to the `Record` type, defaulting to `false`
- Add a `personalNote` text field to the `Record` type, nullable, storing free-form private text
- Expose `setFavorite` and `setPersonalNote` mutations (or extend `updateRecord`) to set these fields
- Allow filtering/sorting the collection by `favorite` status
- Display and edit both fields in the collection UI

## Capabilities

### New Capabilities

- `record-personal-data`: Favorite flag and personal note on records — user-centric, private metadata that travels with the record but is never shared or exported

### Modified Capabilities

- `collection-management`: `Record` type gains two new optional fields (`favorite`, `personalNote`); `updateRecord` input extended accordingly
- `collection-ui`: UI must expose favorite toggle and personal note editor per record

## Impact

- **Database**: `records` table requires two new columns — `favorite BOOLEAN NOT NULL DEFAULT 0` and `personal_note TEXT`
- **GraphQL schema**: `Record` type and `UpdateRecordInput` extended; new filter option on `records` query
- **sqlc queries**: updated `records` queries to include new columns
- **UI**: record card/detail components need favorite icon and note textarea
