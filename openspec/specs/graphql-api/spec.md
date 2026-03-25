## ADDED Requirements

### Requirement: GraphQL HTTP endpoint
The system SHALL expose a GraphQL endpoint at `POST /graphql` accepting `application/json` requests.

#### Scenario: Valid query execution
- **WHEN** an authenticated client sends a valid GraphQL query to `POST /graphql`
- **THEN** the system SHALL return `200 OK` with a JSON body containing `data` and/or `errors`

#### Scenario: Malformed GraphQL request
- **WHEN** a request body is not valid JSON or does not contain a `query` field
- **THEN** the system SHALL return `400 Bad Request` with a descriptive error message

#### Scenario: GraphQL introspection
- **WHEN** a client sends an introspection query
- **THEN** the system SHALL return the full schema description in production mode only if `VYNILINO_INTROSPECTION=true` (default: false in production)

### Requirement: GraphQL Playground / Explorer
The system SHALL serve a GraphQL Playground (or GraphiQL) at `GET /graphql` when `VYNILINO_PLAYGROUND=true` (default: true in development, false in production).

#### Scenario: Playground served in development
- **WHEN** `VYNILINO_PLAYGROUND=true` and a browser requests `GET /graphql`
- **THEN** the system SHALL serve the interactive GraphQL explorer UI

#### Scenario: Playground disabled in production
- **WHEN** `VYNILINO_PLAYGROUND=false`
- **THEN** the system SHALL return `404 Not Found` for `GET /graphql`

### Requirement: WebSocket subscriptions
The system SHALL support GraphQL subscriptions over WebSocket using the `graphql-ws` protocol at `GET /graphql` (upgrade).

#### Scenario: Subscription connection established
- **WHEN** a client initiates a WebSocket upgrade to `/graphql` with the `graphql-transport-ws` subprotocol and a valid token
- **THEN** the system SHALL accept the connection and await subscription operations

#### Scenario: Subscription delivers events
- **WHEN** a subscribed client's collection changes (record added, updated, or deleted)
- **THEN** the system SHALL push the updated record or deletion event to all active subscribers for that user

#### Scenario: Unauthenticated subscription rejected
- **WHEN** a WebSocket connection is made without a valid token in the connection init payload
- **THEN** the system SHALL close the connection with code 4401 (Unauthorized)

### Requirement: Request size limits
The system SHALL enforce a maximum GraphQL request body size of 1MB to prevent abuse.

#### Scenario: Oversized request rejected
- **WHEN** a request body exceeds 1MB
- **THEN** the system SHALL return `413 Request Entity Too Large` before parsing the GraphQL document

### Requirement: Query depth and complexity limits
The system SHALL enforce a maximum query depth of 10 and a complexity budget of 1000 to prevent abusive queries.

#### Scenario: Deep query rejected
- **WHEN** a GraphQL query nests fields beyond depth 10
- **THEN** the system SHALL return a QUERY_TOO_COMPLEX error without executing the query

### Requirement: CORS configuration
The system SHALL support configurable CORS with `VYNILINO_ALLOWED_ORIGINS` (default: `http://localhost:*` for development).

#### Scenario: Allowed origin request
- **WHEN** a request arrives from an origin matching `VYNILINO_ALLOWED_ORIGINS`
- **THEN** the system SHALL respond with appropriate CORS headers

#### Scenario: Disallowed origin blocked
- **WHEN** a request arrives from an origin not in `VYNILINO_ALLOWED_ORIGINS`
- **THEN** the system SHALL return `403 Forbidden` for preflight and omit CORS headers for simple requests

### Requirement: Static asset serving and SPA fallback
The system SHALL serve the embedded web UI static assets and fall back to `index.html` for all non-API, non-asset GET requests, enabling client-side SPA routing.

#### Scenario: Static files served from embed.FS
- **WHEN** a GET request matches a file path present in the embedded `ui/dist/` directory
- **THEN** the system SHALL serve that file with appropriate `Content-Type` and cache headers

#### Scenario: SPA fallback does not intercept API routes
- **WHEN** a GET request path begins with `/graphql`, `/api/`, or `/auth/`
- **THEN** the system SHALL NOT apply the SPA fallback and SHALL route to the appropriate handler

#### Scenario: SPA fallback for unknown paths
- **WHEN** a GET request path does not match any registered route or static asset
- **THEN** the system SHALL return `200 OK` with `ui/dist/index.html`
