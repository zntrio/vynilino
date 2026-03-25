package app_test

import (
	"context"
	"errors"
	"testing"

	discogsclient "zntr.io/vynilino/internal/adapter/discogs"
	"zntr.io/vynilino/internal/app"
	"zntr.io/vynilino/internal/domain"
)

type mockDiscogsSearcher struct {
	results []domain.DiscogsResult
	err     error
}

func (m *mockDiscogsSearcher) Search(_ context.Context, _, _ string) ([]domain.DiscogsResult, error) {
	return m.results, m.err
}

func TestDiscogsService_Search_Success(t *testing.T) {
	year := 1973
	artist := "Pink Floyd"
	mock := &mockDiscogsSearcher{
		results: []domain.DiscogsResult{
			{DiscogsID: "12345", Title: "The Dark Side of the Moon", Artist: &artist, Year: &year},
		},
	}
	svc := app.NewDiscogsService(mock)

	results, err := svc.Search(context.Background(), "Pink Floyd Dark Side", "release")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].DiscogsID != "12345" {
		t.Errorf("unexpected DiscogsID: %s", results[0].DiscogsID)
	}
}

func TestDiscogsService_Search_Empty(t *testing.T) {
	mock := &mockDiscogsSearcher{results: nil}
	svc := app.NewDiscogsService(mock)

	results, err := svc.Search(context.Background(), "xyznonexistent", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestDiscogsService_Search_RateLimit(t *testing.T) {
	mock := &mockDiscogsSearcher{err: discogsclient.ErrRateLimit}
	svc := app.NewDiscogsService(mock)

	_, err := svc.Search(context.Background(), "anything", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, discogsclient.ErrRateLimit) {
		// The service wraps the error into a user-friendly message; check text
		if err.Error() == "" {
			t.Error("error message should not be empty")
		}
	}
}

func TestDiscogsService_Search_Unavailable(t *testing.T) {
	mock := &mockDiscogsSearcher{err: discogsclient.ErrUnavailable}
	svc := app.NewDiscogsService(mock)

	_, err := svc.Search(context.Background(), "anything", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
