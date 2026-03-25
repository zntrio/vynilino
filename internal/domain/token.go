package domain

import (
	"context"
	"time"
)

// RefreshToken represents a stored refresh token.
type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	Revoked   bool
	CreatedAt time.Time
}

// TokenRepository defines persistence operations for refresh tokens.
type TokenRepository interface {
	Create(ctx context.Context, t *RefreshToken) (*RefreshToken, error)
	GetByHash(ctx context.Context, hash string) (*RefreshToken, error)
	Revoke(ctx context.Context, id string) error
	RevokeAllForUser(ctx context.Context, userID string) error
	DeleteExpired(ctx context.Context, before time.Time) error
}
