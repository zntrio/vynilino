package sqlite_test

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	"zntr.io/vynilino/internal/adapter/storage/sqlite"
	"zntr.io/vynilino/internal/domain"
)

func newTestDB(t *testing.T) *testRepos {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "vynilino-test-*.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	f.Close()

	db, err := sqlite.Open(t.Context(), f.Name(), time.Minute)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	return &testRepos{
		users:          sqlite.NewUserRepository(db),
		records:        sqlite.NewRecordRepository(db),
		tokens:         sqlite.NewTokenRepository(db),
		oidcIdentities: sqlite.NewOIDCIdentityRepository(db),
		oidcStates:     sqlite.NewOIDCStateRepository(db),
	}
}

type testRepos struct {
	users          domain.UserRepository
	records        domain.RecordRepository
	tokens         domain.TokenRepository
	oidcIdentities domain.OIDCIdentityRepository
	oidcStates     domain.OIDCStateRepository
}

// ─── User repository ──────────────────────────────────────────────────────────

func TestUserRepo_CreateAndGet(t *testing.T) {
	r := newTestDB(t)
	ctx := context.Background()

	u, err := r.users.Create(ctx, &domain.User{
		Email:        "alice@example.com",
		PasswordHash: "hash",
		Role:         domain.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if u.ID == "" {
		t.Fatal("expected non-empty ID")
	}

	got, err := r.users.GetByEmail(ctx, "alice@example.com")
	if err != nil {
		t.Fatalf("GetByEmail: %v", err)
	}
	if got.ID != u.ID {
		t.Fatalf("ID mismatch: %q != %q", got.ID, u.ID)
	}
}

func TestUserRepo_DuplicateEmail(t *testing.T) {
	r := newTestDB(t)
	ctx := context.Background()
	_, _ = r.users.Create(ctx, &domain.User{Email: "alice@example.com", PasswordHash: "h", Role: domain.RoleUser})
	_, err := r.users.Create(ctx, &domain.User{Email: "alice@example.com", PasswordHash: "h", Role: domain.RoleUser})
	if err == nil {
		t.Fatal("expected error for duplicate email")
	}
}

func TestUserRepo_LoginFailure(t *testing.T) {
	r := newTestDB(t)
	ctx := context.Background()
	u, _ := r.users.Create(ctx, &domain.User{Email: "alice@example.com", PasswordHash: "h", Role: domain.RoleUser})

	lockUntil := time.Now().Add(time.Minute)
	if err := r.users.RecordLoginFailure(ctx, u.ID, &lockUntil); err != nil {
		t.Fatalf("RecordLoginFailure: %v", err)
	}
	got, _ := r.users.GetByID(ctx, u.ID)
	if got.FailedLoginCount != 1 {
		t.Fatalf("expected FailedLoginCount=1, got %d", got.FailedLoginCount)
	}
	if got.LockedUntil == nil {
		t.Fatal("expected LockedUntil to be set")
	}

	if err := r.users.ResetLoginFailure(ctx, u.ID); err != nil {
		t.Fatalf("ResetLoginFailure: %v", err)
	}
	got, _ = r.users.GetByID(ctx, u.ID)
	if got.FailedLoginCount != 0 {
		t.Fatalf("expected FailedLoginCount=0 after reset, got %d", got.FailedLoginCount)
	}
}

func TestUserRepo_UpdatePassword(t *testing.T) {
	r := newTestDB(t)
	ctx := context.Background()

	_, err := r.users.Create(ctx, &domain.User{Email: "alice@example.com", PasswordHash: "oldhash", Role: domain.RoleUser})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := r.users.UpdatePassword(ctx, "alice@example.com", "newhash"); err != nil {
		t.Fatalf("UpdatePassword: %v", err)
	}

	got, err := r.users.GetByEmail(ctx, "alice@example.com")
	if err != nil {
		t.Fatalf("GetByEmail: %v", err)
	}
	if got.PasswordHash != "newhash" {
		t.Fatalf("expected PasswordHash=newhash, got %q", got.PasswordHash)
	}
}

func TestUserRepo_UpdatePassword_NotFound(t *testing.T) {
	r := newTestDB(t)
	ctx := context.Background()

	err := r.users.UpdatePassword(ctx, "nobody@example.com", "hash")
	if err != domain.ErrNotFound {
		t.Fatalf("expected ErrNotFound for unknown email, got %v", err)
	}
}

// ─── Record repository ────────────────────────────────────────────────────────

func TestRecordRepo_CRUD(t *testing.T) {
	r := newTestDB(t)
	ctx := context.Background()
	u, _ := r.users.Create(ctx, &domain.User{Email: "alice@example.com", PasswordHash: "h", Role: domain.RoleUser})

	rec, err := r.records.Create(ctx, &domain.Record{UserID: u.ID, Title: "Dark Side", Artist: "Pink Floyd"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if rec.ID == "" {
		t.Fatal("expected non-empty record ID")
	}

	got, err := r.records.GetByID(ctx, rec.ID, u.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Title != "Dark Side" {
		t.Fatalf("Title mismatch: %q", got.Title)
	}

	got.Title = "The Dark Side of the Moon"
	updated, err := r.records.Update(ctx, got)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Title != "The Dark Side of the Moon" {
		t.Fatalf("Title not updated: %q", updated.Title)
	}

	if err := r.records.Delete(ctx, rec.ID, u.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := r.records.GetByID(ctx, rec.ID, u.ID); err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestRecordRepo_List_Pagination(t *testing.T) {
	r := newTestDB(t)
	ctx := context.Background()
	u, _ := r.users.Create(ctx, &domain.User{Email: "alice@example.com", PasswordHash: "h", Role: domain.RoleUser})

	for i := 0; i < 5; i++ {
		_, _ = r.records.Create(ctx, &domain.Record{UserID: u.ID, Title: "Album", Artist: "Artist"})
	}

	page, err := r.records.List(ctx, u.ID, domain.RecordFilter{Sort: domain.SortByCreatedAt, Direction: domain.SortDesc}, domain.Page{Limit: 3, Offset: 0})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.Records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(page.Records))
	}
	if page.TotalCount != 5 {
		t.Fatalf("expected TotalCount=5, got %d", page.TotalCount)
	}
}

func TestRecordRepo_DiscogsID(t *testing.T) {
	r := newTestDB(t)
	ctx := context.Background()
	u, _ := r.users.Create(ctx, &domain.User{Email: "alice@example.com", PasswordHash: "h", Role: domain.RoleUser})

	discogsID := "12345"
	rec, err := r.records.Create(ctx, &domain.Record{
		UserID:    u.ID,
		Title:     "The Dark Side of the Moon",
		Artist:    "Pink Floyd",
		DiscogsID: &discogsID,
	})
	if err != nil {
		t.Fatalf("Create with DiscogsID: %v", err)
	}
	if rec.DiscogsID == nil || *rec.DiscogsID != discogsID {
		t.Fatalf("DiscogsID not persisted: got %v", rec.DiscogsID)
	}

	got, err := r.records.GetByID(ctx, rec.ID, u.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.DiscogsID == nil || *got.DiscogsID != discogsID {
		t.Fatalf("DiscogsID not returned by GetByID: got %v", got.DiscogsID)
	}

	// Record without DiscogsID should have nil field.
	rec2, err := r.records.Create(ctx, &domain.Record{
		UserID: u.ID,
		Title:  "Manual Entry",
		Artist: "Some Artist",
	})
	if err != nil {
		t.Fatalf("Create without DiscogsID: %v", err)
	}
	if rec2.DiscogsID != nil {
		t.Fatalf("expected nil DiscogsID, got %v", rec2.DiscogsID)
	}

	// Verify DiscogsID appears in list results.
	page, err := r.records.List(ctx, u.ID, domain.RecordFilter{Sort: domain.SortByCreatedAt, Direction: domain.SortDesc}, domain.Page{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	foundDiscogs := false
	for _, listed := range page.Records {
		if listed.ID == rec.ID {
			if listed.DiscogsID == nil || *listed.DiscogsID != discogsID {
				t.Errorf("DiscogsID not returned by List: got %v", listed.DiscogsID)
			}
			foundDiscogs = true
		}
	}
	if !foundDiscogs {
		t.Fatal("record with DiscogsID not found in list")
	}
}

// ─── OIDC Identity repository ─────────────────────────────────────────────────

func TestOIDCIdentityRepo_CreateAndFind(t *testing.T) {
	r := newTestDB(t)
	ctx := context.Background()
	u, _ := r.users.Create(ctx, &domain.User{Email: "alice@example.com", PasswordHash: "h", Role: domain.RoleUser})

	err := r.oidcIdentities.Create(ctx, &domain.OIDCIdentity{
		UserID:   u.ID,
		Provider: "https://accounts.google.com",
		Subject:  "google-sub-123",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	identity, err := r.oidcIdentities.FindByProviderSubject(ctx, "https://accounts.google.com", "google-sub-123")
	if err != nil {
		t.Fatalf("FindByProviderSubject: %v", err)
	}
	if identity.UserID != u.ID {
		t.Fatalf("UserID mismatch: %q", identity.UserID)
	}
}

func TestOIDCIdentityRepo_NotFound(t *testing.T) {
	r := newTestDB(t)
	ctx := context.Background()
	_, err := r.oidcIdentities.FindByProviderSubject(ctx, "https://example.com", "no-such-sub")
	if err != domain.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// ─── OIDC State repository ────────────────────────────────────────────────────

func TestOIDCStateRepo_CreateAndFind(t *testing.T) {
	r := newTestDB(t)
	ctx := context.Background()

	err := r.oidcStates.Create(ctx, &domain.OIDCState{
		State:        "state-abc",
		Nonce:        "nonce-xyz",
		CodeVerifier: "verifier-123",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	s, err := r.oidcStates.FindByState(ctx, "state-abc")
	if err != nil {
		t.Fatalf("FindByState: %v", err)
	}
	if s.Nonce != "nonce-xyz" {
		t.Fatalf("Nonce mismatch: %q", s.Nonce)
	}
}

func TestOIDCStateRepo_DeleteExpired(t *testing.T) {
	r := newTestDB(t)
	ctx := context.Background()

	_ = r.oidcStates.Create(ctx, &domain.OIDCState{
		State:        "old-state",
		Nonce:        "n",
		CodeVerifier: "v",
		CreatedAt:    time.Now().Add(-10 * time.Minute),
	})

	// DeleteExpired with a cutoff 5 minutes ago should remove it.
	if err := r.oidcStates.DeleteExpired(ctx, time.Now().Add(-5*time.Minute)); err != nil {
		t.Fatalf("DeleteExpired: %v", err)
	}

	_, err := r.oidcStates.FindByState(ctx, "old-state")
	if err != domain.ErrNotFound {
		t.Fatalf("expected ErrNotFound after expiry deletion, got %v", err)
	}
}

// ─── Token repository ─────────────────────────────────────────────────────────

func TestTokenRepo_CreateAndRevoke(t *testing.T) {
	r := newTestDB(t)
	ctx := context.Background()
	u, _ := r.users.Create(ctx, &domain.User{Email: "alice@example.com", PasswordHash: "h", Role: domain.RoleUser})

	tok, err := r.tokens.Create(ctx, &domain.RefreshToken{
		UserID:    u.ID,
		TokenHash: "abc123hash",
		ExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := r.tokens.GetByHash(ctx, "abc123hash")
	if err != nil {
		t.Fatalf("GetByHash: %v", err)
	}
	if got.UserID != u.ID {
		t.Fatalf("UserID mismatch")
	}

	if err := r.tokens.Revoke(ctx, tok.ID); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	got2, err := r.tokens.GetByHash(ctx, "abc123hash")
	if err == nil && !got2.Revoked {
		t.Fatal("expected token to be revoked")
	}
}

// ─── Record personal data ─────────────────────────────────────────────────────

func TestRecordRepo_Favorite(t *testing.T) {
	r := newTestDB(t)
	ctx := context.Background()
	u, _ := r.users.Create(ctx, &domain.User{Email: "alice@example.com", PasswordHash: "h", Role: domain.RoleUser})

	rec, err := r.records.Create(ctx, &domain.Record{UserID: u.ID, Title: "Wish You Were Here", Artist: "Pink Floyd"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if rec.Favorite == nil || *rec.Favorite {
		t.Fatal("expected Favorite to be false by default")
	}

	favTrue := true
	rec.Favorite = &favTrue
	updated, err := r.records.Update(ctx, rec)
	if err != nil {
		t.Fatalf("Update (set favorite): %v", err)
	}
	if updated.Favorite == nil || !*updated.Favorite {
		t.Fatal("expected Favorite to be true after update")
	}

	favFalse := false
	rec.Favorite = &favFalse
	updated, err = r.records.Update(ctx, rec)
	if err != nil {
		t.Fatalf("Update (unset favorite): %v", err)
	}
	if updated.Favorite == nil || *updated.Favorite {
		t.Fatal("expected Favorite to be false after unset")
	}
}

func TestRecordRepo_PersonalNote(t *testing.T) {
	r := newTestDB(t)
	ctx := context.Background()
	u, _ := r.users.Create(ctx, &domain.User{Email: "alice@example.com", PasswordHash: "h", Role: domain.RoleUser})

	rec, err := r.records.Create(ctx, &domain.Record{UserID: u.ID, Title: "Animals", Artist: "Pink Floyd"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if rec.PersonalNote != nil {
		t.Fatal("expected PersonalNote to be nil by default")
	}

	note := "Got this from grandma"
	rec.PersonalNote = &note
	updated, err := r.records.Update(ctx, rec)
	if err != nil {
		t.Fatalf("Update (set note): %v", err)
	}
	if updated.PersonalNote == nil || *updated.PersonalNote != note {
		t.Fatalf("expected PersonalNote=%q, got %v", note, updated.PersonalNote)
	}

	empty := ""
	rec.PersonalNote = &empty
	updated, err = r.records.Update(ctx, rec)
	if err != nil {
		t.Fatalf("Update (clear note): %v", err)
	}
	if updated.PersonalNote == nil || *updated.PersonalNote != "" {
		t.Fatalf("expected PersonalNote to be empty string, got %v", updated.PersonalNote)
	}
}

func TestRecordRepo_FavoritesOnlyFilter(t *testing.T) {
	r := newTestDB(t)
	ctx := context.Background()
	u, _ := r.users.Create(ctx, &domain.User{Email: "alice@example.com", PasswordHash: "h", Role: domain.RoleUser})

	fav := true
	_, _ = r.records.Create(ctx, &domain.Record{UserID: u.ID, Title: "Meddle", Artist: "Pink Floyd", Favorite: &fav})
	_, _ = r.records.Create(ctx, &domain.Record{UserID: u.ID, Title: "Obscured by Clouds", Artist: "Pink Floyd"})
	_, _ = r.records.Create(ctx, &domain.Record{UserID: u.ID, Title: "Ummagumma", Artist: "Pink Floyd", Favorite: &fav})

	favPage, err := r.records.List(ctx, u.ID, domain.RecordFilter{FavoritesOnly: true, Sort: domain.SortByCreatedAt, Direction: domain.SortDesc}, domain.Page{Limit: 20})
	if err != nil {
		t.Fatalf("List (favoritesOnly): %v", err)
	}
	if len(favPage.Records) != 2 {
		t.Fatalf("expected 2 favorites, got %d", len(favPage.Records))
	}
	if favPage.TotalCount != 2 {
		t.Fatalf("expected TotalCount=2, got %d", favPage.TotalCount)
	}

	allPage, err := r.records.List(ctx, u.ID, domain.RecordFilter{Sort: domain.SortByCreatedAt, Direction: domain.SortDesc}, domain.Page{Limit: 20})
	if err != nil {
		t.Fatalf("List (all): %v", err)
	}
	if len(allPage.Records) != 3 {
		t.Fatalf("expected 3 records total, got %d", len(allPage.Records))
	}
}

func TestOpen_DirtyMigration(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "vynilino-dirty-*.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	f.Close()

	// Seed a dirty migration state directly, bypassing sqlite.Open.
	raw, err := sql.Open("sqlite", f.Name())
	if err != nil {
		t.Fatalf("open raw db: %v", err)
	}
	_, err = raw.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (version uint64, dirty bool)`)
	if err != nil {
		raw.Close()
		t.Fatalf("create schema_migrations: %v", err)
	}
	_, err = raw.Exec(`INSERT INTO schema_migrations (version, dirty) VALUES (5, 1)`)
	if err != nil {
		raw.Close()
		t.Fatalf("insert dirty state: %v", err)
	}
	raw.Close()

	_, err = sqlite.Open(t.Context(), f.Name(), time.Minute)
	if err == nil {
		t.Fatal("expected error for dirty migration state, got nil")
	}
	if !strings.Contains(err.Error(), "dirty") {
		t.Errorf("error should mention dirty state, got: %v", err)
	}
}
