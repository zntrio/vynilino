package graph

import (
	"context"
	"testing"

	discogsclient "zntr.io/vynilino/internal/adapter/discogs"
	"zntr.io/vynilino/internal/app"
	"zntr.io/vynilino/internal/ctxutil"
	"zntr.io/vynilino/internal/domain"
)

// mockDiscogsSearcher stubs the DiscogsSearcher interface.
type mockDiscogsSearcher struct {
	results []domain.DiscogsResult
	err     error
}

func (m *mockDiscogsSearcher) Search(_ context.Context, _, _ string) ([]domain.DiscogsResult, error) {
	return m.results, m.err
}

func TestSearchDiscogs_Unauthenticated(t *testing.T) {
	svc := app.NewDiscogsService(&mockDiscogsSearcher{})
	r := &Resolver{DiscogsSvc: svc}

	// context without user ID
	_, err := r.Query().SearchDiscogs(context.Background(), "Pink Floyd", nil)
	if err == nil {
		t.Fatal("expected error for unauthenticated caller, got nil")
	}
}

func TestSearchDiscogs_Success(t *testing.T) {
	year := 1973
	artist := "Pink Floyd"
	mock := &mockDiscogsSearcher{
		results: []domain.DiscogsResult{
			{DiscogsID: "42", Title: "Dark Side", Artist: &artist, Year: &year},
		},
	}
	svc := app.NewDiscogsService(mock)
	r := &Resolver{DiscogsSvc: svc}

	ctx := ctxutil.WithUserID(context.Background(), "user-1")
	results, err := r.Query().SearchDiscogs(ctx, "Pink Floyd", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].DiscogsID != "42" {
		t.Errorf("unexpected DiscogsID: %s", results[0].DiscogsID)
	}
}

func TestSearchDiscogs_RateLimit(t *testing.T) {
	mock := &mockDiscogsSearcher{err: discogsclient.ErrRateLimit}
	svc := app.NewDiscogsService(mock)
	r := &Resolver{DiscogsSvc: svc}

	ctx := ctxutil.WithUserID(context.Background(), "user-1")
	_, err := r.Query().SearchDiscogs(ctx, "anything", nil)
	if err == nil {
		t.Fatal("expected error on rate limit, got nil")
	}
}
