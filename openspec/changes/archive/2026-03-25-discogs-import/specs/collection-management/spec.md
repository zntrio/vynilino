## ADDED Requirements

### Requirement: Store Discogs ID on a record
The system SHALL persist an optional `discogs_id` field on each record. This field SHALL be set when a record is created via the Discogs import path and SHALL be null for manually created records.

#### Scenario: Record created with Discogs ID
- **WHEN** an authenticated user submits a `createRecord` mutation with a non-null `discogsId` field
- **THEN** the system SHALL persist the value in the `discogs_id` column of the `records` table and return it in the `Record.discogsId` field

#### Scenario: Record created without Discogs ID
- **WHEN** an authenticated user submits a `createRecord` mutation without the `discogsId` field (manual creation)
- **THEN** the system SHALL persist the record with `discogs_id = NULL` and return `null` for `Record.discogsId`

## MODIFIED Requirements

### Requirement: Add a vinyl record to the collection
The system SHALL allow an authenticated user to add a vinyl record with the following fields: title (required), artist (required), year (optional), label (optional), format (LP/EP/Single/7"/10"/12"), condition (Mint/Near Mint/Very Good Plus/Very Good/Good Plus/Good/Fair/Poor per Goldmine standard), genre (optional, multiple), notes (optional), cover art (optional), discogsId (optional).

#### Scenario: Successful record creation
- **WHEN** an authenticated user submits a valid `createRecord` mutation with title and artist
- **THEN** the system SHALL persist the record, assign a unique ID, set `createdAt` and `updatedAt` timestamps, and return the created record

#### Scenario: Duplicate detection
- **WHEN** a record with the same title and artist already exists in the user's collection
- **THEN** the system SHALL return a warning (not an error) indicating a potential duplicate, and still allow creation if the user confirms

#### Scenario: Missing required fields
- **WHEN** a `createRecord` mutation is submitted without title or artist
- **THEN** the system SHALL return a GraphQL validation error and not persist the record

#### Scenario: Successful record creation from Discogs import
- **WHEN** an authenticated user submits a `createRecord` mutation with a valid `discogsId` and pre-filled metadata from a Discogs search result
- **THEN** the system SHALL persist the record with all provided fields including `discogs_id`, and return the complete record with `discogsId` populated
