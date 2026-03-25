## ADDED Requirements

### Requirement: Mark a record as favorite
The system SHALL allow an authenticated user to mark or unmark any record they own as a favorite by setting the `favorite` field via the `updateRecord` mutation.

#### Scenario: Mark as favorite
- **WHEN** an authenticated user submits `updateRecord` with `favorite: true` for a record they own
- **THEN** the system SHALL persist `favorite = true` on that record and return the updated record with `favorite: true`

#### Scenario: Unmark as favorite
- **WHEN** an authenticated user submits `updateRecord` with `favorite: false` for a record they own
- **THEN** the system SHALL persist `favorite = false` and return the updated record with `favorite: false`

#### Scenario: Default value on creation
- **WHEN** a new record is created without specifying `favorite`
- **THEN** the system SHALL set `favorite` to `false` by default

### Requirement: Attach a personal note to a record
The system SHALL allow an authenticated user to store a free-form personal note on any record they own. The note is private and SHALL NOT be visible to other users.

#### Scenario: Set personal note
- **WHEN** an authenticated user submits `updateRecord` with a non-empty `personalNote` string for a record they own
- **THEN** the system SHALL persist the note and return it in `Record.personalNote`

#### Scenario: Clear personal note
- **WHEN** an authenticated user submits `updateRecord` with `personalNote: ""` (empty string) or `personalNote: null`
- **THEN** the system SHALL set the note to null and return `null` in `Record.personalNote`

#### Scenario: Personal note not accessible by other users
- **WHEN** any query or mutation references a record owned by a different user
- **THEN** the system SHALL NOT return `personalNote` data belonging to that other user (enforced by existing user-scoping on `records`)

### Requirement: Filter collection by favorites
The system SHALL allow an authenticated user to list only their favorited records by passing `favoritesOnly: true` in the `records` query filter.

#### Scenario: Filter returns only favorites
- **WHEN** an authenticated user queries `records(filter: { favoritesOnly: true })`
- **THEN** the system SHALL return only records owned by that user where `favorite = true`, using the same pagination contract as the unfiltered query

#### Scenario: Filter disabled returns all records
- **WHEN** an authenticated user queries `records` without `favoritesOnly` or with `favoritesOnly: false`
- **THEN** the system SHALL return all records owned by that user regardless of favorite status
