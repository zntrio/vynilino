## 1. Database Migration

- [x] 1.1 Create `000005_record_personal_data.up.sql` adding `favorite INTEGER NOT NULL DEFAULT 0` and `personal_note TEXT` columns to `records` table
- [x] 1.2 Create `000005_record_personal_data.down.sql` to reverse the migration (recreate table without new columns)

## 2. sqlc & Data Layer

- [x] 2.1 Update `internal/adapter/storage/sqlite/queries/records.sql` to include `favorite` and `personal_note` in all SELECT, INSERT, and UPDATE queries
- [x] 2.2 Regenerate sqlc code (`sqlc generate`) and verify `sqlcdb/models.go` includes the new fields
- [x] 2.3 Update `internal/adapter/storage/sqlite/record_repo.go` to map `favorite` and `personal_note` to/from the domain `Record` struct
- [x] 2.4 Add `Favorite bool` and `PersonalNote *string` fields to `internal/domain/record.go`
- [x] 2.5 Update `internal/adapter/storage/sqlite/queries/records.sql` `ListRecords` query to support optional `favoritesOnly` filter

## 3. GraphQL Schema & Resolvers

- [x] 3.1 Add `favorite: Boolean!` and `personalNote: String` fields to the `Record` type in `schema.graphql`
- [x] 3.2 Add `favorite: Boolean` and `personalNote: String` to `UpdateRecordInput` in `schema.graphql`
- [x] 3.3 Add `favoritesOnly: Boolean` to `RecordFilter` input (or create it if absent) in `schema.graphql`
- [x] 3.4 Regenerate gqlgen code (`go generate ./...`) and ensure `models_gen.go` reflects the new fields
- [x] 3.5 Update `schema.resolvers.go` — `UpdateRecord` resolver to pass `favorite` and `personalNote` to the app layer
- [x] 3.6 Update `schema.resolvers.go` — `Records` resolver to forward `favoritesOnly` filter to the app layer

## 4. Application Layer

- [x] 4.1 Update `internal/app/record.go` — `UpdateRecord` function to accept and persist `favorite` and `personalNote`
- [x] 4.2 Update `internal/app/record.go` — `ListRecords` function to accept and apply `favoritesOnly` filter

## 5. Integration Tests

- [x] 5.1 Add test in `internal/adapter/storage/sqlite/integration_test.go` covering set/unset favorite on a record
- [x] 5.2 Add test covering set/clear personal note on a record
- [x] 5.3 Add test covering `ListRecords` with `favoritesOnly = true` returns only favorited records

## 6. UI — Favorite Toggle

- [x] 6.1 Add a favorite icon button to the record card component in `ui/src/views/CollectionList.js`
- [x] 6.2 Wire the icon to call `updateRecord` mutation with toggled `favorite` value and apply optimistic update
- [x] 6.3 Add "Favorites only" toggle to the filter panel in `ui/src/views/CollectionList.js`
- [x] 6.4 Pass `favoritesOnly` to the `records` GraphQL query when the toggle is active and handle empty-state message

## 7. UI — Personal Note Editor

- [x] 7.1 Add `personalNote` textarea to the record edit form in `ui/src/views/EditRecord.js` (and `ui/src/components/RecordForm.js` if shared)
- [x] 7.2 Pre-populate the textarea from `record.personalNote` when editing an existing record
- [x] 7.3 Include `personalNote` in the `updateRecord` mutation payload on form submit (send `null` when textarea is empty)

## 8. GraphQL Query Updates

- [x] 8.1 Update the `records` and `record` queries in `ui/src/lib/gql.js` to include `favorite` and `personalNote` fields in the selection set
