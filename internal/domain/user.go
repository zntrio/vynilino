package domain

import (
	"context"
	"errors"
	"time"
)

// Role represents a user's access level.
type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

// User represents an authenticated user.
type User struct {
	ID                string
	Email             string
	PasswordHash      string
	Role              Role
	Active            bool
	FailedLoginCount  int
	LockedUntil       *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// IsLocked reports whether the account is currently locked.
func (u *User) IsLocked(now time.Time) bool {
	return u.LockedUntil != nil && u.LockedUntil.After(now)
}

// UserRepository defines persistence operations for users.
type UserRepository interface {
	Create(ctx context.Context, u *User) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Count(ctx context.Context) (int, error)
	RecordLoginFailure(ctx context.Context, userID string, lockUntil *time.Time) error
	ResetLoginFailure(ctx context.Context, userID string) error
	ListAll(ctx context.Context) ([]*User, error)
	DeactivateUser(ctx context.Context, email string) error
	ActivateUser(ctx context.Context, email string) error
	UpdatePassword(ctx context.Context, email, passwordHash string) error
}

// OIDCIdentity links a vynilino user to an external OIDC provider identity.
type OIDCIdentity struct {
	UserID    string
	Provider  string // issuer URL
	Subject   string // sub claim
	CreatedAt time.Time
}

// OIDCIdentityRepository defines persistence operations for OIDC identities.
type OIDCIdentityRepository interface {
	FindByProviderSubject(ctx context.Context, provider, subject string) (*OIDCIdentity, error)
	Create(ctx context.Context, identity *OIDCIdentity) error
}

// Common domain errors.
var (
	ErrNotFound           = errors.New("not found")
	ErrEmailTaken         = errors.New("email already in use")
	ErrWeakPassword       = errors.New("password does not meet complexity requirements")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountLocked      = errors.New("account is temporarily locked")
	ErrAccountDisabled    = errors.New("account is disabled")
	ErrRegistrationClosed = errors.New("registration is closed")
	ErrInvalidToken       = errors.New("invalid or expired token")
)
