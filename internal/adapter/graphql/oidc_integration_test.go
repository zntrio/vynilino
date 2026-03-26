package graphql

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"zntr.io/vynilino/internal/app"
	"zntr.io/vynilino/internal/ctxutil"
	"zntr.io/vynilino/internal/domain"
)

// testSymKeyHex is a 32-byte key encoded as hex, used only in tests.
const testSymKeyHex = "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"

// ── minimal in-memory repos ───────────────────────────────────────────────────

type minUserRepo struct {
	mu      sync.Mutex
	users   map[string]*domain.User
	byEmail map[string]*domain.User
}

func newMinUserRepo() *minUserRepo {
	return &minUserRepo{users: map[string]*domain.User{}, byEmail: map[string]*domain.User{}}
}

func (r *minUserRepo) Create(_ context.Context, u *domain.User) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
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
func (r *minUserRepo) GetByID(_ context.Context, id string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}
func (r *minUserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.byEmail[email]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}
func (r *minUserRepo) Count(_ context.Context) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.users), nil
}
func (r *minUserRepo) RecordLoginFailure(_ context.Context, _ string, _ *time.Time) error { return nil }
func (r *minUserRepo) ResetLoginFailure(_ context.Context, _ string) error                { return nil }
func (r *minUserRepo) ListAll(_ context.Context) ([]*domain.User, error)                  { return nil, nil }
func (r *minUserRepo) DeactivateUser(_ context.Context, _ string) error                   { return nil }
func (r *minUserRepo) ActivateUser(_ context.Context, _ string) error                     { return nil }
func (r *minUserRepo) UpdatePassword(_ context.Context, _, _ string) error                { return nil }

type minTokenRepo struct {
	mu     sync.Mutex
	tokens map[string]*domain.RefreshToken
	byHash map[string]*domain.RefreshToken
	nextID int
}

func newMinTokenRepo() *minTokenRepo {
	return &minTokenRepo{tokens: map[string]*domain.RefreshToken{}, byHash: map[string]*domain.RefreshToken{}}
}

func (r *minTokenRepo) Create(_ context.Context, t *domain.RefreshToken) (*domain.RefreshToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	t.ID = fmt.Sprintf("%d", r.nextID)
	r.tokens[t.ID] = t
	r.byHash[t.TokenHash] = t
	return t, nil
}
func (r *minTokenRepo) GetByHash(_ context.Context, hash string) (*domain.RefreshToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.byHash[hash]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return t, nil
}
func (r *minTokenRepo) Revoke(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if t, ok := r.tokens[id]; ok {
		t.Revoked = true
	}
	return nil
}
func (r *minTokenRepo) RevokeAllForUser(_ context.Context, _ string) error { return nil }
func (r *minTokenRepo) DeleteExpired(_ context.Context, _ time.Time) error { return nil }

func newTestUserSvc(t *testing.T) *app.UserService {
	t.Helper()
	svc, err := app.NewUserService(newMinUserRepo(), newMinTokenRepo(), testSymKeyHex, "", false, "")
	if err != nil {
		t.Fatalf("NewUserService: %v", err)
	}
	return svc
}

// ── integration test ──────────────────────────────────────────────────────────

// TestOIDCCallback_CookieIsValidatedByAuthMiddleware verifies the full
// end-to-end SSO session path:
//
//  1. The OIDC callback handler sets a vynilino_access cookie whose value is a
//     real PASETO token produced by UserService.
//  2. A subsequent request carrying that cookie is authenticated by
//     AuthMiddleware, i.e. the user ID is injected into the request context.
func TestOIDCCallback_CookieIsValidatedByAuthMiddleware(t *testing.T) {
	userSvc := newTestUserSvc(t)

	// Register a user that the OIDC flow would have resolved.
	pair, err := userSvc.Register(context.Background(), "sso@example.com", "TestPassword1!")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Simulate HandleCallback returning the real token pair.
	svc := &stubCallbackSvc{pair: pair}
	h := oidcCallbackHandler(svc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oidc/callback?code=abc&state=xyz", nil)
	h(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("callback: want 302, got %d", rec.Code)
	}

	var cookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == oidcCookieName {
			cookie = c
			break
		}
	}
	if cookie == nil {
		t.Fatalf("expected %q cookie to be set", oidcCookieName)
	}

	// A follow-up request carrying the cookie must be authenticated.
	var capturedUserID string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid, ok := ctxutil.UserIDFromContext(r.Context())
		if !ok {
			t.Error("no user ID in context: SSO cookie not recognized by AuthMiddleware")
			return
		}
		capturedUserID = uid
	})

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.AddCookie(cookie)
	rr2 := httptest.NewRecorder()
	AuthMiddleware(userSvc, userSvc)(inner).ServeHTTP(rr2, req2)

	if capturedUserID == "" {
		t.Fatal("SSO session cookie was rejected by AuthMiddleware")
	}
}
