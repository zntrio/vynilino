## ADDED Requirements

### Requirement: Favorite toggle on record card
The UI SHALL display a favorite toggle (e.g. a heart or star icon) on each record card in the collection list. Clicking it SHALL immediately toggle the `favorite` field via an `updateRecord` mutation.

#### Scenario: Toggle favorite on
- **WHEN** an authenticated user clicks the favorite icon on a record where `favorite` is `false`
- **THEN** the UI SHALL submit `updateRecord` with `favorite: true`, optimistically render the icon as active, and revert only if the mutation fails

#### Scenario: Toggle favorite off
- **WHEN** an authenticated user clicks the favorite icon on a record where `favorite` is `true`
- **THEN** the UI SHALL submit `updateRecord` with `favorite: false`, optimistically render the icon as inactive, and revert only if the mutation fails

#### Scenario: Favorite status persists on reload
- **WHEN** the user reloads the page
- **THEN** all records with `favorite: true` SHALL render with the active favorite icon

### Requirement: Personal note editor on record detail
The UI SHALL display a personal note textarea on the record edit form and record detail view. Changes SHALL be saved via `updateRecord`.

#### Scenario: Display personal note in edit form
- **WHEN** an authenticated user opens the edit form for a record that has a non-null `personalNote`
- **THEN** the textarea SHALL be pre-populated with the existing note text

#### Scenario: Save personal note
- **WHEN** an authenticated user edits the personal note textarea and submits the form
- **THEN** the UI SHALL include the `personalNote` value in the `updateRecord` mutation and display the saved note after success

#### Scenario: Clear personal note
- **WHEN** an authenticated user clears the personal note textarea and submits the form
- **THEN** the UI SHALL send `personalNote: null` (or empty string) in `updateRecord` and the note field SHALL display as empty after save

### Requirement: Filter collection by favorites
The UI SHALL provide a "Favorites only" toggle in the search/filter panel that, when active, restricts the collection list to favorited records.

#### Scenario: Activate favorites filter
- **WHEN** an authenticated user activates the "Favorites only" toggle
- **THEN** the UI SHALL re-execute the `records` query with `filter: { favoritesOnly: true }` and update the list to show only favorited records

#### Scenario: Deactivate favorites filter
- **WHEN** the user deactivates the "Favorites only" toggle
- **THEN** the UI SHALL re-execute the `records` query without the `favoritesOnly` filter and restore the full collection list

#### Scenario: Empty favorites state
- **WHEN** the user activates the "Favorites only" filter and has no favorited records
- **THEN** the UI SHALL display an empty-state message such as "No favorites yet — click the heart on any record to add it here"
