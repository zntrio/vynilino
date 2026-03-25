## ADDED Requirements

### Requirement: Per-request nonce generation
The system SHALL generate a cryptographically random, base64url-encoded nonce of at least 128 bits of entropy for every HTML entry-point response (`GET /`, `GET /login`, and SPA fallback paths).

#### Scenario: Nonce generated per request
- **WHEN** the server handles a request for an HTML entry point
- **THEN** the server SHALL generate a new nonce value unique to that response before writing any headers or body

#### Scenario: Nonce entropy
- **WHEN** a nonce is generated
- **THEN** it SHALL be produced from a cryptographically secure random source and contain at least 16 bytes (128 bits) of entropy encoded as base64url

### Requirement: Content-Security-Policy header with nonce directives
The system SHALL set a `Content-Security-Policy` response header on every HTML entry-point response that restricts script and style execution to resources bearing the per-request nonce.

#### Scenario: CSP header present on HTML responses
- **WHEN** the server returns an HTML entry-point response
- **THEN** the response SHALL include a `Content-Security-Policy` header containing at least the following directives:
  - `script-src 'nonce-<value>'`
  - `style-src 'nonce-<value>'`
  where `<value>` is the nonce generated for that request

#### Scenario: CSP header absent on non-HTML responses
- **WHEN** the server returns a non-HTML response (e.g. a static asset, a GraphQL response, or an API JSON response)
- **THEN** the response SHALL NOT include a `Content-Security-Policy` header injected by the nonce middleware

#### Scenario: Inline scripts without nonce are blocked
- **WHEN** a browser receives an HTML entry-point response with the CSP header
- **THEN** any inline `<script>` or `<style>` element that does not carry the matching `nonce` attribute SHALL be blocked by the browser's CSP enforcement

### Requirement: Nonce propagation via request context
The system SHALL store the generated nonce in the request context so that any handler or template that renders the HTML entry point can read it without regenerating a new value.

#### Scenario: Nonce readable from context in handler
- **WHEN** a nonce has been placed in the request context by the nonce middleware
- **THEN** the HTML-serving handler SHALL retrieve the same nonce value from the context and use it when rendering the response

#### Scenario: Absence of nonce in context causes error
- **WHEN** the HTML-serving handler cannot find a nonce in the request context (e.g. middleware was not applied)
- **THEN** the handler SHALL return HTTP 500 and SHALL NOT serve the HTML page without a nonce

### Requirement: Nonce injection into HTML entry point
The system SHALL inject the nonce value as an attribute on every `<script>` and `<link rel="stylesheet">` element within the HTML entry-point template before the response body is written.

#### Scenario: Script tags carry nonce attribute
- **WHEN** the server renders `index.html` or `login.html` for a request
- **THEN** every `<script>` element in the served HTML SHALL have a `nonce="<value>"` attribute matching the per-request nonce

#### Scenario: Stylesheet link tags carry nonce attribute
- **WHEN** the server renders `index.html` or `login.html` for a request
- **THEN** every `<link rel="stylesheet">` element in the served HTML SHALL have a `nonce="<value>"` attribute matching the per-request nonce

#### Scenario: Two requests receive different nonces
- **WHEN** the server handles two successive requests for the same HTML entry point
- **THEN** the nonce values injected into each response SHALL be different

### Requirement: Cache-Control: no-store on HTML entry-point responses
The system SHALL set `Cache-Control: no-store` on every HTML entry-point response to prevent intermediate caches or the browser cache from storing a page that contains a nonce.

#### Scenario: HTML entry point response is not cached
- **WHEN** the server returns an HTML entry-point response
- **THEN** the response SHALL include the header `Cache-Control: no-store`

#### Scenario: Static assets are not affected
- **WHEN** the server returns a versioned static asset (e.g. `/assets/main.abc123.js`)
- **THEN** the `Cache-Control` header for that asset SHALL remain `public, max-age=31536000, immutable` and SHALL NOT be overridden to `no-store`
