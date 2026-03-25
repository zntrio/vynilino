## ADDED Requirements

### Requirement: Serve embedded web UI
The system SHALL serve a single-page web application at `GET /` (and all non-API sub-paths) from static assets embedded in the Go binary at build time via `//go:embed ui/dist`.

#### Scenario: Root path returns index.html
- **WHEN** an unauthenticated or authenticated browser requests `GET /`
- **THEN** the system SHALL return `200 OK` with `Content-Type: text/html` and the contents of `ui/dist/index.html`

#### Scenario: Static assets served with cache headers
- **WHEN** a browser requests a hashed asset such as `GET /assets/main.abc123.js`
- **THEN** the system SHALL return `200 OK` with `Cache-Control: public, max-age=31536000, immutable`

#### Scenario: SPA fallback for deep links
- **WHEN** a browser requests any path under `/` that is not `/graphql`, `/api/`, `/auth/`, or a known static asset
- **THEN** the system SHALL return `200 OK` with the contents of `ui/dist/index.html` to support client-side routing

### Requirement: Current user identity endpoint
The system SHALL expose `GET /api/me` returning the authenticated user's profile so the UI can determine login state without a GraphQL query.

#### Scenario: Authenticated user
- **WHEN** a request to `GET /api/me` is made with a valid session cookie
- **THEN** the system SHALL return `200 OK` with a JSON body `{"id": "<id>", "email": "<email>", "name": "<name>"}`

#### Scenario: Unauthenticated user
- **WHEN** a request to `GET /api/me` is made without a valid session
- **THEN** the system SHALL return `401 Unauthorized` with body `{"error": "unauthenticated"}`

### Requirement: Collection list view
The UI SHALL display a paginated list of the authenticated user's vinyl records with title, artist, year, format, condition, and cover art thumbnail.

#### Scenario: Records displayed on load
- **WHEN** an authenticated user navigates to `/`
- **THEN** the UI SHALL fetch the first page of records via GraphQL `records` query and render each as a card or table row

#### Scenario: Load more pagination
- **WHEN** the user clicks "Load more" and `hasNextPage` is true
- **THEN** the UI SHALL fetch the next page using the `endCursor` and append results to the list without a full page reload

#### Scenario: Empty collection state
- **WHEN** the authenticated user has no records
- **THEN** the UI SHALL display an empty-state illustration and a prominent "Add your first record" call-to-action button

#### Scenario: Real-time update on collection change
- **WHEN** a record is added, updated, or deleted (by any client) and the UI has an active WebSocket subscription
- **THEN** the UI SHALL reflect the change in the list within 2 seconds without requiring a manual refresh

### Requirement: Add record form
The UI SHALL provide a form to create a new vinyl record with all fields defined in the collection-management spec.

#### Scenario: Successful record creation
- **WHEN** an authenticated user fills in the required fields (title, artist) and submits the "Add Record" form
- **THEN** the UI SHALL submit a `createRecord` GraphQL mutation, close the form, and display the new record in the list

#### Scenario: Duplicate warning displayed
- **WHEN** the API returns a duplicate warning alongside the created record
- **THEN** the UI SHALL display an inline warning banner informing the user of the potential duplicate, while still showing the new record

#### Scenario: Validation error displayed
- **WHEN** the user submits the form with missing required fields
- **THEN** the UI SHALL display field-level error messages and NOT submit the mutation

### Requirement: Edit record form
The UI SHALL allow an authenticated user to edit any field of an existing record via an edit form pre-populated with current values.

#### Scenario: Successful update
- **WHEN** an authenticated user modifies fields and submits the edit form
- **THEN** the UI SHALL submit an `updateRecord` mutation and reflect the updated values in the list and detail view

#### Scenario: Optimistic UI update
- **WHEN** the user submits the edit form
- **THEN** the UI SHALL immediately render the updated values optimistically and revert only if the mutation returns an error

### Requirement: Delete record confirmation
The UI SHALL require explicit confirmation before deleting a record.

#### Scenario: Delete confirmed
- **WHEN** an authenticated user clicks "Delete" and confirms in the confirmation dialog
- **THEN** the UI SHALL submit a `deleteRecord` mutation and remove the record from the list

#### Scenario: Delete cancelled
- **WHEN** an authenticated user clicks "Delete" but dismisses the confirmation dialog
- **THEN** the UI SHALL NOT submit any mutation and the record SHALL remain in the list

### Requirement: Cover art upload
The UI SHALL allow an authenticated user to upload a cover art image when creating or editing a record.

#### Scenario: Image upload via file input
- **WHEN** a user selects an image file (JPEG, PNG, WebP ≤ 5 MB) in the cover art field
- **THEN** the UI SHALL POST the file to `POST /api/upload` and store the returned URL in the form state for submission with the record mutation

#### Scenario: Oversized file rejected client-side
- **WHEN** a user selects a file larger than 5 MB
- **THEN** the UI SHALL display an error message and NOT initiate the upload

### Requirement: Search and filter panel
The UI SHALL provide a search bar and filter controls that query the backend in real time.

#### Scenario: Text search
- **WHEN** the user types in the search bar (debounced 300 ms)
- **THEN** the UI SHALL re-execute the `records` query with the `query` argument and update the list

#### Scenario: Format filter
- **WHEN** the user selects one or more formats from the filter panel
- **THEN** the UI SHALL re-execute the `records` query filtered by the selected formats

#### Scenario: Clear all filters
- **WHEN** the user clicks "Clear filters"
- **THEN** the UI SHALL reset all filter state and reload the full collection list

### Requirement: Responsive layout
The UI SHALL adapt its layout for desktop (viewport ≥ 768 px) and mobile (viewport < 768 px).

#### Scenario: Desktop sidebar navigation
- **WHEN** the viewport width is ≥ 768 px
- **THEN** the UI SHALL render a persistent left sidebar with navigation links and collection statistics

#### Scenario: Mobile bottom navigation
- **WHEN** the viewport width is < 768 px
- **THEN** the UI SHALL render a bottom navigation bar and hide the sidebar

#### Scenario: Mobile add record sheet
- **WHEN** a mobile user taps the "Add Record" button
- **THEN** the UI SHALL open the add form in a bottom sheet modal occupying full viewport width

### Requirement: OIDC login and logout
The UI SHALL integrate with the existing OIDC redirect-based authentication flow.

#### Scenario: Unauthenticated redirect
- **WHEN** the UI detects a `401` response from `GET /api/me` on load
- **THEN** the UI SHALL redirect the browser to `GET /auth/login`

#### Scenario: Logout
- **WHEN** an authenticated user clicks "Logout"
- **THEN** the UI SHALL navigate to `GET /auth/logout`, which invalidates the session server-side and redirects back to the login page
