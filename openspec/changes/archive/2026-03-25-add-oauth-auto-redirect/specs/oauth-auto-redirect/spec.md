## ADDED Requirements

### Requirement: OIDC auto-redirect configuration flag
The system SHALL support a `VYNILINO_OIDC_AUTO_REDIRECT` environment variable (boolean, default `false`) that, when set to `true` together with a configured OIDC provider, causes `GET /login` to respond with an HTTP 302 redirect to the OIDC authorization URL without sending the login page HTML.

#### Scenario: Flag disabled by default
- **WHEN** `VYNILINO_OIDC_AUTO_REDIRECT` is not set
- **THEN** `GET /login` SHALL serve the login page HTML as normal

#### Scenario: Flag enabled without OIDC configured
- **WHEN** `VYNILINO_OIDC_AUTO_REDIRECT=true` and `VYNILINO_OIDC_ISSUER` is not set
- **THEN** `GET /login` SHALL serve the login page HTML as normal (OIDC is inactive)

### Requirement: Server-side redirect on GET /login
When auto-redirect is active, the server SHALL generate a fresh OIDC authorization URL and redirect the client before any login page content is transmitted.

#### Scenario: Successful redirect
- **WHEN** `VYNILINO_OIDC_AUTO_REDIRECT=true` and `VYNILINO_OIDC_ISSUER` is set
- **AND** a client issues `GET /login`
- **THEN** the server SHALL call `OIDCService.AuthorizationURL` to generate a PKCE-bound URL with a new `state`
- **AND** SHALL respond with HTTP 302 and a `Location` header pointing to the OIDC provider authorization endpoint
- **AND** SHALL NOT send any login page HTML in the response body

#### Scenario: OIDC provider unreachable
- **WHEN** `VYNILINO_OIDC_AUTO_REDIRECT=true` and `VYNILINO_OIDC_ISSUER` is set
- **AND** `OIDCService.AuthorizationURL` returns an error (provider unreachable or misconfigured)
- **THEN** the server SHALL respond with HTTP 500
- **AND** SHALL return a minimal HTML error page indicating the OIDC provider is unavailable and the user should contact their administrator
- **AND** SHALL NOT serve the login form as a fallback

#### Scenario: No fallback on error
- **WHEN** auto-redirect is active and the authorization URL cannot be generated
- **THEN** the server SHALL NOT fall back to rendering the login page
- **AND** the only recovery path is an operator action: restore the OIDC provider or unset `VYNILINO_OIDC_AUTO_REDIRECT`
