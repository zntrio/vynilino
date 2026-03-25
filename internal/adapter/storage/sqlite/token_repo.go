package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"zntr.io/vynilino/internal/adapter/storage/sqlite/sqlcdb"
	"zntr.io/vynilino/internal/domain"
)

type tokenRepository struct {
	q *sqlcdb.Queries
}

// NewTokenRepository returns a domain.TokenRepository backed by SQLite.
func NewTokenRepository(db *sql.DB) domain.TokenRepository {
	return &tokenRepository{q: sqlcdb.New(db)}
}

func (r *tokenRepository) Create(ctx context.Context, t *domain.RefreshToken) (*domain.RefreshToken, error) {
	now := time.Now()
	t.ID = uuid.NewString()
	t.CreatedAt = now

	row, err := r.q.CreateRefreshToken(ctx, sqlcdb.CreateRefreshTokenParams{
		ID:        t.ID,
		UserID:    t.UserID,
		TokenHash: t.TokenHash,
		ExpiresAt: t.ExpiresAt.Unix(),
		CreatedAt: now.Unix(),
	})
	if err != nil {
		return nil, err
	}
	return tokenFromRow(row), nil
}

func (r *tokenRepository) GetByHash(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	row, err := r.q.GetRefreshTokenByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return tokenFromRow(row), nil
}

func (r *tokenRepository) Revoke(ctx context.Context, id string) error {
	return r.q.RevokeRefreshToken(ctx, id)
}

func (r *tokenRepository) RevokeAllForUser(ctx context.Context, userID string) error {
	return r.q.RevokeAllUserTokens(ctx, userID)
}

func (r *tokenRepository) DeleteExpired(ctx context.Context, before time.Time) error {
	return r.q.DeleteExpiredTokens(ctx, before.Unix())
}

func tokenFromRow(row sqlcdb.RefreshToken) *domain.RefreshToken {
	return &domain.RefreshToken{
		ID:        row.ID,
		UserID:    row.UserID,
		TokenHash: row.TokenHash,
		ExpiresAt: time.Unix(row.ExpiresAt, 0),
		Revoked:   row.Revoked != 0,
		CreatedAt: time.Unix(row.CreatedAt, 0),
	}
}
