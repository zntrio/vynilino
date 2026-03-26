package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"

	"zntr.io/vynilino/internal/adapter/storage/sqlite/sqlcdb"
	"zntr.io/vynilino/internal/domain"
)

type userRepository struct {
	q *sqlcdb.Queries
}

// NewUserRepository returns a domain.UserRepository backed by SQLite.
func NewUserRepository(db *sql.DB) domain.UserRepository {
	return &userRepository{q: sqlcdb.New(db)}
}

func (r *userRepository) Create(ctx context.Context, u *domain.User) (*domain.User, error) {
	now := time.Now()
	u.ID = uuid.NewString()
	u.CreatedAt = now
	u.UpdatedAt = now

	row, err := r.q.CreateUser(ctx, sqlcdb.CreateUserParams{
		ID:           u.ID,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Role:         string(u.Role),
		CreatedAt:    now.Unix(),
		UpdatedAt:    now.Unix(),
	})
	if err != nil {
		if isDuplicateError(err) {
			return nil, domain.ErrEmailTaken
		}
		return nil, err
	}
	return userFromRow(row), nil
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	row, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return userFromRow(row), nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	row, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return userFromRow(row), nil
}

func (r *userRepository) Count(ctx context.Context) (int, error) {
	n, err := r.q.CountUsers(ctx)
	return int(n), err
}

func (r *userRepository) RecordLoginFailure(ctx context.Context, userID string, lockUntil *time.Time) error {
	var lockUnixNull sql.NullInt64
	if lockUntil != nil {
		lockUnixNull = sql.NullInt64{Int64: lockUntil.Unix(), Valid: true}
	}
	return r.q.UpdateLoginFailure(ctx, sqlcdb.UpdateLoginFailureParams{
		LockedUntil: lockUnixNull,
		UpdatedAt:   time.Now().Unix(),
		ID:          userID,
	})
}

func (r *userRepository) ResetLoginFailure(ctx context.Context, userID string) error {
	return r.q.ResetLoginFailure(ctx, sqlcdb.ResetLoginFailureParams{
		UpdatedAt: time.Now().Unix(),
		ID:        userID,
	})
}

func (r *userRepository) ListAll(ctx context.Context) ([]*domain.User, error) {
	rows, err := r.q.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	users := make([]*domain.User, len(rows))
	for i, row := range rows {
		users[i] = userFromRow(row)
	}
	return users, nil
}

func (r *userRepository) DeactivateUser(ctx context.Context, email string) error {
	return r.q.DeactivateUser(ctx, sqlcdb.DeactivateUserParams{
		UpdatedAt: time.Now().Unix(),
		Email:     email,
	})
}

func (r *userRepository) ActivateUser(ctx context.Context, email string) error {
	return r.q.ActivateUser(ctx, sqlcdb.ActivateUserParams{
		UpdatedAt: time.Now().Unix(),
		Email:     email,
	})
}

func (r *userRepository) UpdatePassword(ctx context.Context, email, passwordHash string) error {
	if _, err := r.GetByEmail(ctx, email); err != nil {
		return err // already domain.ErrNotFound if the email is unknown
	}
	return r.q.ChangeUserPassword(ctx, sqlcdb.ChangeUserPasswordParams{
		PasswordHash: passwordHash,
		UpdatedAt:    time.Now().Unix(),
		Email:        email,
	})
}

// --- helpers ---

func userFromRow(row sqlcdb.User) *domain.User {
	u := &domain.User{
		ID:               row.ID,
		Email:            row.Email,
		PasswordHash:     row.PasswordHash,
		Role:             domain.Role(row.Role),
		Active:           row.Active != 0,
		FailedLoginCount: int(row.FailedLoginCount),
		CreatedAt:        time.Unix(row.CreatedAt, 0),
		UpdatedAt:        time.Unix(row.UpdatedAt, 0),
	}
	if row.LockedUntil.Valid {
		t := time.Unix(row.LockedUntil.Int64, 0)
		u.LockedUntil = &t
	}
	return u
}

func isDuplicateError(err error) bool {
	return err != nil && (containsStr(err.Error(), "UNIQUE constraint failed") ||
		containsStr(err.Error(), "unique constraint"))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// jsonStrings marshals a string slice to a JSON string for storage.
func jsonStrings(ss []string) string {
	if len(ss) == 0 {
		return "[]"
	}
	b, _ := json.Marshal(ss)
	return string(b)
}

// parseJSONStrings parses a JSON string array.
func parseJSONStrings(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	_ = json.Unmarshal([]byte(s), &out)
	return out
}
