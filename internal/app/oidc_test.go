package app_test

import (
	"context"
	"testing"
	"time"

	"zntr.io/vynilino/internal/app"
	"zntr.io/vynilino/internal/config"
	"zntr.io/vynilino/internal/domain"
)

// ─── In-memory stubs ──────────────────────────────────────────────────────────

type memOIDCIdentityRepo struct {
	entries map[string]*domain.OIDCIdentity // key: provider+"|"+subject
}

func newMemOIDCIdentityRepo() *memOIDCIdentityRepo {
	return &memOIDCIdentityRepo{entries: map[string]*domain.OIDCIdentity{}}
}

func (r *memOIDCIdentityRepo) key(provider, subject string) string {
	return provider + "|" + subject
}

func (r *memOIDCIdentityRepo) FindByProviderSubject(_ context.Context, provider, subject string) (*domain.OIDCIdentity, error) {
	if e, ok := r.entries[r.key(provider, subject)]; ok {
		return e, nil
	}
	return nil, domain.ErrNotFound
}

func (r *memOIDCIdentityRepo) Create(_ context.Context, id *domain.OIDCIdentity) error {
	r.entries[r.key(id.Provider, id.Subject)] = id
	return nil
}

type memOIDCStateRepo struct {
	states map[string]*domain.OIDCState
}

func newMemOIDCStateRepo() *memOIDCStateRepo {
	return &memOIDCStateRepo{states: map[string]*domain.OIDCState{}}
}

func (r *memOIDCStateRepo) Create(_ context.Context, s *domain.OIDCState) error {
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now()
	}
	r.states[s.State] = s
	return nil
}

func (r *memOIDCStateRepo) FindByState(_ context.Context, state string) (*domain.OIDCState, error) {
	if s, ok := r.states[state]; ok {
		return s, nil
	}
	return nil, domain.ErrNotFound
}

func (r *memOIDCStateRepo) Delete(_ context.Context, state string) error {
	delete(r.states, state)
	return nil
}

func (r *memOIDCStateRepo) DeleteExpired(_ context.Context, olderThan time.Time) error {
	for k, s := range r.states {
		if s.CreatedAt.Before(olderThan) {
			delete(r.states, k)
		}
	}
	return nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func newOIDCService(t *testing.T, cfg *config.Config, userSvc *app.UserService) (*app.OIDCService, *memOIDCIdentityRepo, *memOIDCStateRepo) {
	t.Helper()
	identityRepo := newMemOIDCIdentityRepo()
	stateRepo := newMemOIDCStateRepo()
	svc, err := app.NewOIDCService(cfg, userSvc, identityRepo, stateRepo)
	if err != nil {
		t.Fatalf("NewOIDCService: %v", err)
	}
	return svc, identityRepo, stateRepo
}

func testOIDCConfig() *config.Config {
	return &config.Config{
		TokenKey:         testKeyHex,
		OIDCIssuer:       "https://accounts.example.com",
		OIDCClientID:     "client-id",
		OIDCClientSecret: "client-secret",
		OIDCRedirectURL:  "http://localhost:8080/oidc/callback",
	}
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestOIDCService_DisabledWhenNoIssuer(t *testing.T) {
	users := newMemUserRepo()
	tokens := newMemTokenRepo()
	userSvc, _ := app.NewUserService(users, tokens, testKeyHex, "", false, "")

	cfg := &config.Config{TokenKey: testKeyHex, OIDCIssuer: ""}
	svc, _, _ := newOIDCService(t, cfg, userSvc)
	if svc != nil {
		t.Fatal("expected nil OIDCService when issuer is empty")
	}
}

func TestOIDCService_StateExpiry(t *testing.T) {
	users := newMemUserRepo()
	tokens := newMemTokenRepo()
	userSvc, _ := app.NewUserService(users, tokens, testKeyHex, "", false, "")

	cfg := testOIDCConfig()
	svc, _, stateRepo := newOIDCService(t, cfg, userSvc)
	if svc == nil {
		t.Skip("OIDC not configured")
	}

	// Insert an expired state directly into the repo.
	expired := &domain.OIDCState{
		State:        "expired-state",
		Nonce:        "n",
		CodeVerifier: "v",
		CreatedAt:    time.Now().Add(-10 * time.Minute),
	}
	_ = stateRepo.Create(context.Background(), expired)

	_, err := svc.HandleCallback(context.Background(), "any-code", "expired-state")
	if err != domain.ErrOIDCStateExpired {
		t.Fatalf("expected ErrOIDCStateExpired, got %v", err)
	}
}

func TestOIDCService_InvalidState(t *testing.T) {
	users := newMemUserRepo()
	tokens := newMemTokenRepo()
	userSvc, _ := app.NewUserService(users, tokens, testKeyHex, "", false, "")

	cfg := testOIDCConfig()
	svc, _, _ := newOIDCService(t, cfg, userSvc)
	if svc == nil {
		t.Skip("OIDC not configured")
	}

	_, err := svc.HandleCallback(context.Background(), "code", "no-such-state")
	if err != domain.ErrOIDCInvalidState {
		t.Fatalf("expected ErrOIDCInvalidState, got %v", err)
	}
}

func TestOIDCService_RegistrationClosedInSingleOwnerMode(t *testing.T) {
	users := newMemUserRepo()
	tokens := newMemTokenRepo()
	// Create one existing user so single-owner blocks provisioning.
	_, _ = users.Create(context.Background(), &domain.User{Email: "existing@example.com", PasswordHash: "h", Role: domain.RoleAdmin})

	userSvc, _ := app.NewUserService(users, tokens, testKeyHex, "", true /* singleOwner */, "")

	cfg := testOIDCConfig()
	svc, _, stateRepo := newOIDCService(t, cfg, userSvc)
	if svc == nil {
		t.Skip("OIDC not configured")
	}

	// Inject a valid (non-expired) state.
	_ = stateRepo.Create(context.Background(), &domain.OIDCState{
		State:        "s",
		Nonce:        "n",
		CodeVerifier: "v",
	})

	// HandleCallback will fail at OIDC token exchange (no real provider),
	// but we specifically want to test registration-closed; so we test
	// resolveUser indirectly by calling the exported method and checking
	// that a NOT ErrOIDCTokenInvalid path leads to ErrRegistrationClosed.
	// Since we can't mock the real HTTP exchange, we call a test-visible
	// helper via exported func (see oidc_export_test.go).
	err := svc.ResolveUserForTest(context.Background(), "https://accounts.example.com", "sub-new", "new@example.com", true)
	if err != domain.ErrRegistrationClosed {
		t.Fatalf("expected ErrRegistrationClosed, got %v", err)
	}
}

func TestOIDCService_UnverifiedEmailSkipsLinking(t *testing.T) {
	users := newMemUserRepo()
	tokens := newMemTokenRepo()
	// Create a user with the email we'll try to link.
	_, _ = users.Create(context.Background(), &domain.User{Email: "alice@example.com", PasswordHash: "h", Role: domain.RoleAdmin})

	userSvc, _ := app.NewUserService(users, tokens, testKeyHex, "", false, "")

	cfg := testOIDCConfig()
	svc, identityRepo, _ := newOIDCService(t, cfg, userSvc)
	if svc == nil {
		t.Skip("OIDC not configured")
	}

	// emailVerified=false → should NOT link, should provision a new account.
	err := svc.ResolveUserForTest(context.Background(), "https://accounts.example.com", "sub-xyz", "alice@example.com", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The identity should have been created for the new user, not Alice's user.
	id, err := identityRepo.FindByProviderSubject(context.Background(), "https://accounts.example.com", "sub-xyz")
	if err != nil {
		t.Fatalf("identity not created: %v", err)
	}
	// Verify it's NOT linked to the existing user (alice@example.com → user-alice@example.com).
	if id.UserID == "user-alice@example.com" {
		t.Fatal("unverified email should not link to existing user")
	}
}
