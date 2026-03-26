package domain

import (
	"context"
	"errors"
	"time"
)

// OIDCState holds the server-side state for an in-flight OIDC Authorization
// Code + PKCE flow. Entries expire after a short TTL (enforced at read time).
type OIDCState struct {
	State        string // 32-byte random hex; used as primary key
	Nonce        string // 32-byte random hex; included in ID token claims
	CodeVerifier string // PKCE S256 code verifier
	CreatedAt    time.Time
}

// OIDCStateRepository defines persistence operations for OIDC flow state.
type OIDCStateRepository interface {
	Create(ctx context.Context, s *OIDCState) error
	FindByState(ctx context.Context, state string) (*OIDCState, error)
	Delete(ctx context.Context, state string) error
	DeleteExpired(ctx context.Context, olderThan time.Time) error
}

// OIDC-specific domain errors.
var (
	ErrOIDCNotConfigured = errors.New("OIDC is not configured")
	ErrOIDCInvalidState  = errors.New("invalid or unknown OIDC state")
	ErrOIDCStateExpired  = errors.New("OIDC state has expired")
	ErrOIDCTokenInvalid  = errors.New("OIDC ID token validation failed")
	ErrOIDCUserForbidden = errors.New("SSO login forbidden: user does not exist in the local directory")
)
