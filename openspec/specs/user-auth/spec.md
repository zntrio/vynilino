## ADDED Requirements

### Requirement: User registration
The system SHALL allow new users to register with an email and password when registration is enabled. In single-owner mode (default), only one account SHALL be creatable; subsequent registration attempts SHALL be rejected.

#### Scenario: First user registration (single-owner mode)
- **WHEN** no users exist and a `register` mutation is submitted with valid email and password
- **THEN** the system SHALL create the account, assign the ADMIN role, and return an access token and refresh token

#### Scenario: Registration blocked in single-owner mode
- **WHEN** a user already exists and single-owner mode is active
- **THEN** the system SHALL return a REGISTRATION_CLOSED error

#### Scenario: Weak password rejected
- **WHEN** a registration is submitted with a password shorter than 12 characters or lacking complexity
- **THEN** the system SHALL return a WEAK_PASSWORD error without creating the account

#### Scenario: Duplicate email rejected
- **WHEN** a registration is submitted with an email already in use
- **THEN** the system SHALL return an EMAIL_TAKEN error

### Requirement: User login
The system SHALL authenticate users via email and password, returning a short-lived access token (PASETO v2 local) and a long-lived refresh token.

#### Scenario: Successful login
- **WHEN** a `login` mutation is submitted with valid credentials
- **THEN** the system SHALL return an access token (15-minute TTL) and a refresh token (30-day TTL)

#### Scenario: Invalid credentials
- **WHEN** a `login` mutation is submitted with wrong email or password
- **THEN** the system SHALL return an INVALID_CREDENTIALS error after a constant-time comparison (no timing oracle)

#### Scenario: Account lockout after repeated failures
- **WHEN** 10 consecutive failed login attempts occur for the same email within 15 minutes
- **THEN** the system SHALL lock the account for 15 minutes and return an ACCOUNT_LOCKED error

### Requirement: Token refresh
The system SHALL allow clients to obtain a new access token using a valid refresh token.

#### Scenario: Successful refresh
- **WHEN** a `refreshToken` mutation is called with a valid, non-expired refresh token
- **THEN** the system SHALL return a new access token and rotate the refresh token (old token invalidated)

#### Scenario: Expired or invalid refresh token
- **WHEN** a `refreshToken` mutation is called with an expired or revoked token
- **THEN** the system SHALL return an INVALID_TOKEN error

### Requirement: Logout
The system SHALL allow authenticated users to invalidate their current refresh token.

#### Scenario: Successful logout
- **WHEN** an authenticated user calls the `logout` mutation
- **THEN** the system SHALL revoke the current refresh token and return a success confirmation

### Requirement: Authentication enforcement
All GraphQL queries and mutations (except `login`, `register`, `refreshToken`) SHALL require a valid Bearer access token in the Authorization header.

#### Scenario: Unauthenticated request rejected
- **WHEN** a GraphQL operation is sent without a valid Authorization header
- **THEN** the system SHALL return an UNAUTHENTICATED error (HTTP 401 for the transport layer)

#### Scenario: Expired access token rejected
- **WHEN** a GraphQL operation is sent with an expired access token
- **THEN** the system SHALL return a TOKEN_EXPIRED error so the client can trigger a refresh

### Requirement: Password hashing
The system SHALL store passwords using Argon2id with secure default parameters (memory ≥ 64MB, iterations ≥ 3, parallelism ≥ 2).

#### Scenario: Password never stored in plaintext
- **WHEN** a user registers or changes their password
- **THEN** the system SHALL store only the Argon2id hash; the plaintext password SHALL NOT appear in logs, errors, or database fields

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
