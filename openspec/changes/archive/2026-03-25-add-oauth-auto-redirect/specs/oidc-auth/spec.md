## MODIFIED Requirements

### Requirement: OIDC provider configuration
The system SHALL support a single OIDC provider configured via environment variables. OIDC support SHALL be entirely disabled (no routes, no mutations exposed) when `VYNILINO_OIDC_ISSUER` is unset. The system SHALL additionally support `VYNILINO_OIDC_AUTO_REDIRECT` (boolean, default `false`) which, when `true`, causes the server to redirect `GET /login` directly to the OIDC authorization endpoint without serving the login page HTML.

#### Scenario: OIDC disabled when issuer not configured
- **WHEN** `VYNILINO_OIDC_ISSUER` is not set
- **THEN** the `oidcAuthorizationURL` and `oidcCallback` mutations SHALL return a NOT_CONFIGURED error and no OIDC-related state SHALL be created

#### Scenario: Provider discovery on first use
- **WHEN** `VYNILINO_OIDC_ISSUER` is set and the first OIDC operation is triggered
- **THEN** the system SHALL fetch `<issuer>/.well-known/openid-configuration` and cache the provider metadata (authorization endpoint, token endpoint, JWKS URI)

#### Scenario: Auto-redirect active — login page bypassed
- **WHEN** `VYNILINO_OIDC_ISSUER` is set and `VYNILINO_OIDC_AUTO_REDIRECT=true`
- **AND** a client issues `GET /login`
- **THEN** the server SHALL respond with HTTP 302 to the OIDC authorization URL
- **AND** SHALL NOT serve the login page HTML

#### Scenario: Auto-redirect inactive — login page served normally
- **WHEN** `VYNILINO_OIDC_AUTO_REDIRECT` is `false` (or unset), regardless of `VYNILINO_OIDC_ISSUER`
- **THEN** `GET /login` SHALL serve the login page HTML unchanged
