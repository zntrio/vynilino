## ADDED Requirements

### Requirement: Add a vinyl record to the collection
The system SHALL allow an authenticated user to add a vinyl record with the following fields: title (required), artist (required), year (optional), label (optional), format (LP/EP/Single/7"/10"/12"), condition (Mint/Near Mint/Very Good Plus/Very Good/Good Plus/Good/Fair/Poor per Goldmine standard), genre (optional, multiple), notes (optional), cover art (optional), discogsId (optional), favorite (optional, default false), personalNote (optional).

#### Scenario: Successful record creation
- **WHEN** an authenticated user submits a valid `createRecord` mutation with title and artist
- **THEN** the system SHALL persist the record, assign a unique ID, set `createdAt` and `updatedAt` timestamps, set `favorite` to `false`, set `personalNote` to `null`, and return the created record

#### Scenario: Duplicate detection
- **WHEN** a record with the same title and artist already exists in the user's collection
- **THEN** the system SHALL return a warning (not an error) indicating a potential duplicate, and still allow creation if the user confirms

#### Scenario: Missing required fields
- **WHEN** a `createRecord` mutation is submitted without title or artist
- **THEN** the system SHALL return a GraphQL validation error and not persist the record

#### Scenario: Successful record creation from Discogs import
- **WHEN** an authenticated user submits a `createRecord` mutation with a valid `discogsId` and pre-filled metadata from a Discogs search result
- **THEN** the system SHALL persist the record with all provided fields including `discogs_id`, and return the complete record with `discogsId` populated

### Requirement: Update a vinyl record
The system SHALL allow an authenticated user to update any field of a record they own via an `updateRecord` mutation, including `favorite` (boolean) and `personalNote` (string, nullable).

#### Scenario: Successful update
- **WHEN** an authenticated user submits an `updateRecord` mutation with a valid record ID and changed fields
- **THEN** the system SHALL update only the provided fields, refresh `updatedAt`, and return the updated record

#### Scenario: Update non-existent record
- **WHEN** an `updateRecord` mutation references an ID that does not exist or belongs to another user
- **THEN** the system SHALL return a NOT_FOUND error

#### Scenario: Update favorite field
- **WHEN** an authenticated user submits `updateRecord` with `favorite: true`
- **THEN** the system SHALL persist the new favorite status and return it in the response

#### Scenario: Update personalNote field
- **WHEN** an authenticated user submits `updateRecord` with a non-null `personalNote`
- **THEN** the system SHALL persist the note and return it in `Record.personalNote`

### Requirement: Remove a vinyl record
The system SHALL allow an authenticated user to delete a record from their collection.

#### Scenario: Successful deletion
- **WHEN** an authenticated user submits a `deleteRecord` mutation with a valid record ID
- **THEN** the system SHALL remove the record and its associated cover art, and return a success confirmation

#### Scenario: Delete non-existent record
- **WHEN** a `deleteRecord` mutation references an ID not owned by the user
- **THEN** the system SHALL return a NOT_FOUND error

### Requirement: List collection records
The system SHALL allow an authenticated user to retrieve their collection with pagination support (cursor-based).

#### Scenario: Paginated list
- **WHEN** an authenticated user queries `records` with optional `first` and `after` arguments
- **THEN** the system SHALL return a paginated connection with edges, nodes, and pageInfo (hasNextPage, endCursor)

#### Scenario: Empty collection
- **WHEN** an authenticated user queries `records` and has no records
- **THEN** the system SHALL return an empty connection with `hasNextPage: false`

### Requirement: Retrieve a single record
The system SHALL allow an authenticated user to fetch a specific record by ID.

#### Scenario: Successful fetch
- **WHEN** an authenticated user queries `record(id: "...")` with a valid ID they own
- **THEN** the system SHALL return the full record with all fields

#### Scenario: Record not found
- **WHEN** `record(id: "...")` is called with an unknown or unauthorized ID
- **THEN** the system SHALL return null with a NOT_FOUND error in the errors array

### Requirement: Store Discogs ID on a record
The system SHALL persist an optional `discogs_id` field on each record. This field SHALL be set when a record is created via the Discogs import path and SHALL be null for manually created records.

#### Scenario: Record created with Discogs ID
- **WHEN** an authenticated user submits a `createRecord` mutation with a non-null `discogsId` field
- **THEN** the system SHALL persist the value in the `discogs_id` column of the `records` table and return it in the `Record.discogsId` field

#### Scenario: Record created without Discogs ID
- **WHEN** an authenticated user submits a `createRecord` mutation without the `discogsId` field (manual creation)
- **THEN** the system SHALL persist the record with `discogs_id = NULL` and return `null` for `Record.discogsId`
