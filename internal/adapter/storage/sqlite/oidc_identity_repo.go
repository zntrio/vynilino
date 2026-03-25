package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"zntr.io/vynilino/internal/adapter/storage/sqlite/sqlcdb"
	"zntr.io/vynilino/internal/domain"
)

type oidcIdentityRepository struct {
	q *sqlcdb.Queries
}

// NewOIDCIdentityRepository returns a domain.OIDCIdentityRepository backed by SQLite.
func NewOIDCIdentityRepository(db *sql.DB) domain.OIDCIdentityRepository {
	return &oidcIdentityRepository{q: sqlcdb.New(db)}
}

func (r *oidcIdentityRepository) FindByProviderSubject(ctx context.Context, provider, subject string) (*domain.OIDCIdentity, error) {
	row, err := r.q.FindOIDCIdentityByProviderSubject(ctx, sqlcdb.FindOIDCIdentityByProviderSubjectParams{
		Provider: provider,
		Subject:  subject,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &domain.OIDCIdentity{
		UserID:    row.UserID,
		Provider:  row.Provider,
		Subject:   row.Subject,
		CreatedAt: row.CreatedAt,
	}, nil
}

func (r *oidcIdentityRepository) Create(ctx context.Context, identity *domain.OIDCIdentity) error {
	if identity.CreatedAt.IsZero() {
		identity.CreatedAt = time.Now()
	}
	return r.q.CreateOIDCIdentity(ctx, sqlcdb.CreateOIDCIdentityParams{
		UserID:    identity.UserID,
		Provider:  identity.Provider,
		Subject:   identity.Subject,
		CreatedAt: identity.CreatedAt,
	})
}
