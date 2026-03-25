## ADDED Requirements

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

### Requirement: Authorization URL generation
The system SHALL generate an OIDC Authorization Code Flow URL with PKCE (S256) for the configured provider.

#### Scenario: Successful authorization URL generation
- **WHEN** an unauthenticated client calls `oidcAuthorizationURL`
- **THEN** the system SHALL return a URL containing `response_type=code`, `client_id`, `redirect_uri`, `scope=openid email profile`, a cryptographically random `state` (32-byte hex), a `nonce`, and a `code_challenge` derived from a random `code_verifier` using S256
- **AND** the `state`, `nonce`, and `code_verifier` SHALL be persisted server-side with a 5-minute TTL

#### Scenario: State expires after 5 minutes
- **WHEN** an `oidcCallback` is received with a `state` older than 5 minutes
- **THEN** the system SHALL return an OIDC_STATE_EXPIRED error and reject the callback

### Requirement: Authorization code callback
The system SHALL handle the OIDC callback, exchange the authorization code for tokens, validate the ID token, and issue vynilino tokens.

#### Scenario: Successful callback for new user (auto-provisioning)
- **WHEN** `oidcCallback` is called with a valid `code` and matching `state`
- **AND** no existing vynilino user matches the `sub` claim or the `email` claim
- **AND** single-owner mode is inactive or no user exists yet
- **THEN** the system SHALL exchange the code for an ID token, validate it, create a new vynilino user account (no password), store the OIDC identity link, and return a vynilino access token and refresh token

#### Scenario: Successful callback for existing user via sub claim
- **WHEN** `oidcCallback` is called with a valid `code` and matching `state`
- **AND** an `oidc_identities` row exists for `(provider, sub)`
- **THEN** the system SHALL issue a vynilino access token and refresh token for that user without creating a new account

#### Scenario: Account linking via verified email claim
- **WHEN** `oidcCallback` is called and no OIDC identity exists for `(provider, sub)`
- **AND** the ID token contains an `email_verified=true` claim
- **AND** a vynilino user exists with that email address
- **THEN** the system SHALL create an `oidc_identities` row linking the OIDC subject to that user and return vynilino tokens

#### Scenario: Auto-provisioning blocked in single-owner mode
- **WHEN** `oidcCallback` would create a new user
- **AND** single-owner mode is active and a user already exists
- **THEN** the system SHALL return a REGISTRATION_CLOSED error without creating any account or identity link

#### Scenario: Unverified email not used for account linking
- **WHEN** the ID token contains `email_verified=false` or no `email_verified` claim
- **THEN** the system SHALL NOT attempt to link the OIDC identity to any existing user by email; it SHALL only auto-provision a new account (subject to single-owner rules)

#### Scenario: Invalid state rejected
- **WHEN** `oidcCallback` is called with a `state` that does not match any server-side record
- **THEN** the system SHALL return an OIDC_INVALID_STATE error

#### Scenario: ID token validation failure
- **WHEN** the ID token signature, `iss`, `aud`, `exp`, or `nonce` check fails
- **THEN** the system SHALL return an OIDC_TOKEN_INVALID error without issuing any vynilino tokens

### Requirement: OIDC identity storage
The system SHALL persist the mapping between a vynilino user and an OIDC provider subject in a durable store.

#### Scenario: Identity stored after successful login
- **WHEN** an OIDC callback results in a new identity link (auto-provision or account linking)
- **THEN** the `oidc_identities` table SHALL contain a row with `user_id`, `provider` (issuer URL), `subject` (`sub` claim), and `created_at`

#### Scenario: State cleaned up after use
- **WHEN** an `oidcCallback` is processed (success or failure after validation)
- **THEN** the server-side state record SHALL be deleted to prevent replay
