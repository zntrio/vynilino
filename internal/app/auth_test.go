package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"zntr.io/vynilino/internal/app"
	"zntr.io/vynilino/internal/domain"
)

// ─── In-memory stubs ──────────────────────────────────────────────────────────

type memUserRepo struct {
	users  map[string]*domain.User
	byEmail map[string]*domain.User
}

func newMemUserRepo() *memUserRepo {
	return &memUserRepo{users: map[string]*domain.User{}, byEmail: map[string]*domain.User{}}
}

func (r *memUserRepo) Create(_ context.Context, u *domain.User) (*domain.User, error) {
	if _, ok := r.byEmail[u.Email]; ok {
		return nil, domain.ErrEmailTaken
	}
	u.ID = "user-" + u.Email
	u.Active = true
	u.CreatedAt = time.Now()
	u.UpdatedAt = time.Now()
	r.users[u.ID] = u
	r.byEmail[u.Email] = u
	return u, nil
}
func (r *memUserRepo) GetByID(_ context.Context, id string) (*domain.User, error) {
	u, ok := r.users[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}
func (r *memUserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	u, ok := r.byEmail[email]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}
func (r *memUserRepo) Count(_ context.Context) (int, error)  { return len(r.users), nil }
func (r *memUserRepo) RecordLoginFailure(_ context.Context, userID string, lockUntil *time.Time) error {
	u, ok := r.users[userID]
	if !ok {
		return nil
	}
	u.FailedLoginCount++
	u.LockedUntil = lockUntil
	return nil
}
func (r *memUserRepo) ResetLoginFailure(_ context.Context, userID string) error {
	if u, ok := r.users[userID]; ok {
		u.FailedLoginCount = 0
		u.LockedUntil = nil
	}
	return nil
}
func (r *memUserRepo) ListAll(_ context.Context) ([]*domain.User, error)                     { return nil, nil }
func (r *memUserRepo) DeactivateUser(_ context.Context, email string) error                  { return nil }
func (r *memUserRepo) ActivateUser(_ context.Context, email string) error                    { return nil }
func (r *memUserRepo) UpdatePassword(_ context.Context, email, hash string) error            { return nil }

type memTokenRepo struct {
	tokens  map[string]*domain.RefreshToken // by ID
	byHash  map[string]*domain.RefreshToken
}

func newMemTokenRepo() *memTokenRepo {
	return &memTokenRepo{tokens: map[string]*domain.RefreshToken{}, byHash: map[string]*domain.RefreshToken{}}
}

func (r *memTokenRepo) Create(_ context.Context, t *domain.RefreshToken) (*domain.RefreshToken, error) {
	t.ID = "tok-" + t.TokenHash[:8]
	t.CreatedAt = time.Now()
	r.tokens[t.ID] = t
	r.byHash[t.TokenHash] = t
	return t, nil
}
func (r *memTokenRepo) GetByHash(_ context.Context, hash string) (*domain.RefreshToken, error) {
	t, ok := r.byHash[hash]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return t, nil
}
func (r *memTokenRepo) Revoke(_ context.Context, id string) error {
	if t, ok := r.tokens[id]; ok {
		t.Revoked = true
	}
	return nil
}
func (r *memTokenRepo) RevokeAllForUser(_ context.Context, userID string) error {
	for _, t := range r.tokens {
		if t.UserID == userID {
			t.Revoked = true
		}
	}
	return nil
}
func (r *memTokenRepo) DeleteExpired(_ context.Context, before time.Time) error { return nil }

// ─── Test key (32 hex-encoded bytes) ─────────────────────────────────────────

const testKeyHex = "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"

func newTestUserService(t *testing.T) (*app.UserService, *memUserRepo, *memTokenRepo) {
	t.Helper()
	users := newMemUserRepo()
	tokens := newMemTokenRepo()
	svc, err := app.NewUserService(users, tokens, testKeyHex, true)
	if err != nil {
		t.Fatalf("NewUserService: %v", err)
	}
	return svc, users, tokens
}

// ─── Tests ───────────────────────────────────────────────────────────────────

func TestRegister_Success(t *testing.T) {
	svc, _, _ := newTestUserService(t)
	pair, err := svc.Register(context.Background(), "alice@example.com", "S3cretPass1!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("expected non-empty tokens")
	}
}

func TestRegister_SingleOwnerBlocked(t *testing.T) {
	svc, _, _ := newTestUserService(t)
	_, _ = svc.Register(context.Background(), "alice@example.com", "S3cretPass1!")
	_, err := svc.Register(context.Background(), "bob@example.com", "S3cretPass1!")
	if !errors.Is(err, domain.ErrRegistrationClosed) {
		t.Fatalf("expected ErrRegistrationClosed, got %v", err)
	}
}

func TestRegister_WeakPassword(t *testing.T) {
	svc, _, _ := newTestUserService(t)
	_, err := svc.Register(context.Background(), "alice@example.com", "weak")
	if !errors.Is(err, domain.ErrWeakPassword) {
		t.Fatalf("expected ErrWeakPassword, got %v", err)
	}
}

func TestLogin_Success(t *testing.T) {
	svc, _, _ := newTestUserService(t)
	_, _ = svc.Register(context.Background(), "alice@example.com", "S3cretPass1!")
	pair, err := svc.Login(context.Background(), "alice@example.com", "S3cretPass1!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pair.AccessToken == "" {
		t.Fatal("expected access token")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	svc, _, _ := newTestUserService(t)
	_, _ = svc.Register(context.Background(), "alice@example.com", "S3cretPass1!")
	_, err := svc.Login(context.Background(), "alice@example.com", "WrongPass1!")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_AccountLockout(t *testing.T) {
	svc, users, _ := newTestUserService(t)
	_, _ = svc.Register(context.Background(), "alice@example.com", "S3cretPass1!")

	// Simulate 9 failures already recorded.
	user, _ := users.GetByEmail(context.Background(), "alice@example.com")
	user.FailedLoginCount = 9

	_, err := svc.Login(context.Background(), "alice@example.com", "WrongPass1!")
	if !errors.Is(err, domain.ErrAccountLocked) {
		t.Fatalf("expected ErrAccountLocked on 10th failure, got %v", err)
	}
}

func TestRefreshToken_Rotation(t *testing.T) {
	svc, _, tokens := newTestUserService(t)
	_, _ = svc.Register(context.Background(), "alice@example.com", "S3cretPass1!")
	pair1, _ := svc.Login(context.Background(), "alice@example.com", "S3cretPass1!")

	pair2, err := svc.RefreshToken(context.Background(), pair1.RefreshToken)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pair2.RefreshToken == pair1.RefreshToken {
		t.Fatal("refresh token should be rotated")
	}

	// Old token must now be revoked.
	_, err = svc.RefreshToken(context.Background(), pair1.RefreshToken)
	if !errors.Is(err, domain.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken after rotation, got %v", err)
	}
	_ = tokens
}

func TestValidateAccessToken(t *testing.T) {
	svc, _, _ := newTestUserService(t)
	pair, _ := svc.Register(context.Background(), "alice@example.com", "S3cretPass1!")
	userID, err := svc.ValidateAccessToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userID == "" {
		t.Fatal("expected non-empty userID")
	}
}
