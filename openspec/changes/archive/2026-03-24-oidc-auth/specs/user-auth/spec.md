## ADDED Requirements

### Requirement: OIDC login mutations
The GraphQL API SHALL expose two new mutations for the OIDC flow: `oidcAuthorizationURL` and `oidcCallback`. These mutations SHALL be callable without authentication (unauthenticated), consistent with `login` and `register`.

#### Scenario: oidcAuthorizationURL mutation available when OIDC configured
- **WHEN** `VYNILINO_OIDC_ISSUER` is set and a client calls `oidcAuthorizationURL`
- **THEN** the system SHALL return an `OIDCAuthURLPayload` containing `url` (the full authorization URL to redirect the user to) and `state` (the opaque state value for client-side correlation)

#### Scenario: oidcCallback mutation available when OIDC configured
- **WHEN** `VYNILINO_OIDC_ISSUER` is set and a client calls `oidcCallback(code: "...", state: "...")`
- **THEN** the system SHALL validate the flow and return an `AuthPayload` (same type as `login`) containing `accessToken`, `refreshToken`, and `expiresIn`

#### Scenario: OIDC mutations return error when not configured
- **WHEN** `VYNILINO_OIDC_ISSUER` is not set and a client calls `oidcAuthorizationURL` or `oidcCallback`
- **THEN** the system SHALL return a GraphQL error with code `NOT_CONFIGURED`
