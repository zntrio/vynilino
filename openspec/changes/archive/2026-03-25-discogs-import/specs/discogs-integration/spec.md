## ADDED Requirements

### Requirement: Search Discogs database
The system SHALL provide a GraphQL query `searchDiscogs(query: String!, type: DiscogsSearchType): [DiscogsResult!]!` that proxies requests to the Discogs `/database/search` API and returns a list of matching releases. The query SHALL be authenticated (only available to logged-in users).

#### Scenario: Successful search returns results
- **WHEN** an authenticated user executes `searchDiscogs(query: "Dark Side of the Moon", type: RELEASE)`
- **THEN** the system SHALL return an array of `DiscogsResult` objects each containing `discogsId`, `title`, `artist`, `year`, `label`, `format`, `thumbUrl`, and `country` populated from the Discogs API response

#### Scenario: Empty search results
- **WHEN** an authenticated user executes `searchDiscogs` with a query that matches no Discogs entries
- **THEN** the system SHALL return an empty array with no error

#### Scenario: Discogs API rate limit exceeded
- **WHEN** the Discogs API returns HTTP 429
- **THEN** the system SHALL return a GraphQL error with message "Discogs rate limit exceeded. Please try again shortly." and SHALL NOT crash or return partial data

#### Scenario: Discogs API unreachable
- **WHEN** the Discogs API does not respond within 5 seconds
- **THEN** the system SHALL return a GraphQL error with message "Discogs service unavailable." and the manual record creation path SHALL remain fully operational

#### Scenario: Unauthenticated search rejected
- **WHEN** an unauthenticated request executes `searchDiscogs`
- **THEN** the system SHALL return a GraphQL authentication error and SHALL NOT call the Discogs API

### Requirement: Discogs token configuration
The system SHALL read an optional `VYNILINO_DISCOGS_TOKEN` environment variable. When present, the token SHALL be sent as a `Authorization: Discogs token=<value>` header on all Discogs API requests to increase the rate limit from 25 to 60 requests per minute.

#### Scenario: Authenticated Discogs requests when token is set
- **WHEN** `VYNILINO_DISCOGS_TOKEN` is set and a `searchDiscogs` query is executed
- **THEN** the outbound HTTP request to `api.discogs.com` SHALL include the `Authorization: Discogs token=<value>` header

#### Scenario: Unauthenticated Discogs requests when token is absent
- **WHEN** `VYNILINO_DISCOGS_TOKEN` is not set and a `searchDiscogs` query is executed
- **THEN** the outbound HTTP request SHALL include only the required `User-Agent` header and no `Authorization` header
