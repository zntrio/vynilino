## ADDED Requirements

### Requirement: Serve embedded web UI
The system SHALL serve two separate single-page application shells from static assets embedded in the Go binary at build time via `//go:embed ui/dist`.

- `GET /login` (and `/login.html`) SHALL serve `ui/dist/login.html` — the login-only bundle.
- `GET /` and all non-API, non-asset, non-login sub-paths SHALL serve `ui/dist/index.html` — the authenticated app shell.

#### Scenario: Root path returns index.html
- **WHEN** an authenticated browser requests `GET /`
- **THEN** the system SHALL return `200 OK` with `Content-Type: text/html` and the contents of `ui/dist/index.html`

#### Scenario: Login path returns login.html
- **WHEN** any browser requests `GET /login` or `GET /login.html`
- **THEN** the system SHALL return `200 OK` with `Content-Type: text/html` and the contents of `ui/dist/login.html`

#### Scenario: Static assets served with cache headers
- **WHEN** a browser requests a hashed asset such as `GET /assets/main.abc123.js`
- **THEN** the system SHALL return `200 OK` with `Cache-Control: public, max-age=31536000, immutable`

#### Scenario: SPA fallback for deep links (authenticated app)
- **WHEN** a browser requests any path under `/` that is not `/graphql`, `/api/`, `/auth/`, `/login`, `/login.html`, or a known static asset
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

### Requirement: Login bundle isolation
The Vite build SHALL produce two independent entry points with no shared JavaScript chunks between them:

- `login.html` + `login.js`: loads Alpine.js, the login form component, and OIDC redirect logic only.
- `index.html` + `main.js`: loads the full application (Alpine.js, router, all views).

Neither bundle SHALL import or statically depend on modules from the other entry point.

#### Scenario: Login page loads without application code
- **WHEN** an unauthenticated user navigates to `/login`
- **THEN** the browser SHALL download only the login bundle assets (login.js and its dependencies) and SHALL NOT download main.js or any app-only chunk

#### Scenario: App page loads without login code
- **WHEN** an authenticated user navigates to `/`
- **THEN** the browser SHALL download only the app bundle assets and SHALL NOT download login.js

### Requirement: Login view standalone entry point
The login view (`src/login.js`) SHALL be a self-contained Alpine.js application that:
- Initialises Alpine with only the auth store and toast store.
- Renders the login form (email/password) with submit triggering the GraphQL `login` mutation.
- Renders the OIDC login button (if OIDC is available) triggering the `oidcAuthorizationURL` mutation and redirecting.
- On successful login, stores the access token and redirects to `/`.

#### Scenario: Successful password login redirects to app
- **WHEN** a user submits valid credentials on the login page
- **THEN** the login bundle SHALL store the token in localStorage and navigate the browser to `/`

#### Scenario: Failed login shows error
- **WHEN** a user submits invalid credentials
- **THEN** the login bundle SHALL display an inline error message without redirecting

### Requirement: Discogs search panel in Add Record view
The UI SHALL provide a "Search Discogs" panel within the Add Record view that allows the user to search the Discogs database and pre-populate the record form with the selected result.

#### Scenario: User opens Discogs search panel
- **WHEN** an authenticated user navigates to the Add Record view
- **THEN** the UI SHALL render a "Search Discogs" toggle or tab alongside the manual entry form

#### Scenario: User searches Discogs
- **WHEN** the user types a query (artist name, album title, or barcode) in the Discogs search input and submits
- **THEN** the UI SHALL execute the `searchDiscogs` GraphQL query and display a list of results with cover thumbnail, title, artist, year, label, and format for each result

#### Scenario: User selects a Discogs result
- **WHEN** the user clicks on a result in the Discogs search list
- **THEN** the UI SHALL pre-populate the record creation form fields (title, artist, year, label, format, coverArtUrl, discogsId) with the values from the selected result, and focus the form so the user can review and adjust before saving

#### Scenario: No Discogs results found
- **WHEN** the `searchDiscogs` query returns an empty array
- **THEN** the UI SHALL display a "No results found" message and allow the user to refine their search or switch to manual entry

#### Scenario: Discogs search error displayed
- **WHEN** the `searchDiscogs` query returns a GraphQL error (e.g. rate limit, timeout)
- **THEN** the UI SHALL display a user-friendly error message (e.g. "Discogs search is temporarily unavailable. You can still add records manually.") and keep the manual form accessible

#### Scenario: Pre-populated form still editable
- **WHEN** the user has selected a Discogs result and the form is pre-populated
- **THEN** the user SHALL be able to modify any field before submitting the `createRecord` mutation

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
