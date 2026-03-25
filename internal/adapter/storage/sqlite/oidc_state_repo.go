package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"zntr.io/vynilino/internal/adapter/storage/sqlite/sqlcdb"
	"zntr.io/vynilino/internal/domain"
)

type oidcStateRepository struct {
	q *sqlcdb.Queries
}

// NewOIDCStateRepository returns a domain.OIDCStateRepository backed by SQLite.
func NewOIDCStateRepository(db *sql.DB) domain.OIDCStateRepository {
	return &oidcStateRepository{q: sqlcdb.New(db)}
}

func (r *oidcStateRepository) Create(ctx context.Context, s *domain.OIDCState) error {
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now()
	}
	return r.q.CreateOIDCState(ctx, sqlcdb.CreateOIDCStateParams{
		State:        s.State,
		Nonce:        s.Nonce,
		CodeVerifier: s.CodeVerifier,
		CreatedAt:    s.CreatedAt,
	})
}

func (r *oidcStateRepository) FindByState(ctx context.Context, state string) (*domain.OIDCState, error) {
	row, err := r.q.FindOIDCStateByState(ctx, state)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &domain.OIDCState{
		State:        row.State,
		Nonce:        row.Nonce,
		CodeVerifier: row.CodeVerifier,
		CreatedAt:    row.CreatedAt,
	}, nil
}

func (r *oidcStateRepository) Delete(ctx context.Context, state string) error {
	return r.q.DeleteOIDCState(ctx, state)
}

func (r *oidcStateRepository) DeleteExpired(ctx context.Context, olderThan time.Time) error {
	return r.q.DeleteExpiredOIDCStates(ctx, olderThan)
}
