## ADDED Requirements

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
