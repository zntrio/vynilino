## ADDED Requirements

### Requirement: Full-text search across the collection
The system SHALL support full-text search over record title, artist, label, and notes fields via a `search` query argument.

#### Scenario: Matching search
- **WHEN** an authenticated user queries `records(search: "pink floyd")`
- **THEN** the system SHALL return records whose title or artist contains the search terms (case-insensitive)

#### Scenario: No results
- **WHEN** a search query matches no records
- **THEN** the system SHALL return an empty connection (not an error)

#### Scenario: Empty search string
- **WHEN** `search: ""` is provided
- **THEN** the system SHALL behave as if no search filter was applied and return all records

### Requirement: Filter collection by structured fields
The system SHALL support filtering the collection by one or more of: artist (exact/partial), genre (exact), year range (min/max), format, condition (minimum grade).

#### Scenario: Filter by artist
- **WHEN** an authenticated user queries `records(filter: { artist: "Beatles" })`
- **THEN** the system SHALL return only records where the artist field contains "Beatles" (case-insensitive)

#### Scenario: Filter by year range
- **WHEN** an authenticated user queries `records(filter: { yearMin: 1965, yearMax: 1970 })`
- **THEN** the system SHALL return only records with year between 1965 and 1970 inclusive

#### Scenario: Filter by condition
- **WHEN** an authenticated user queries `records(filter: { conditionMin: VERY_GOOD })`
- **THEN** the system SHALL return only records graded Very Good or above per Goldmine scale ordering

#### Scenario: Combined filters
- **WHEN** multiple filter fields are provided simultaneously
- **THEN** the system SHALL apply all filters as AND conditions

### Requirement: Sort collection results
The system SHALL support sorting records by: title, artist, year, createdAt, updatedAt — ascending or descending.

#### Scenario: Sort by year descending
- **WHEN** an authenticated user queries `records(sort: { field: YEAR, direction: DESC })`
- **THEN** the system SHALL return records ordered from newest to oldest year, with null years last

#### Scenario: Default sort order
- **WHEN** no sort argument is provided
- **THEN** the system SHALL sort by `createdAt` descending (most recently added first)
