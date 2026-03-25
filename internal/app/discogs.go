package app

import (
	"context"
	"errors"
	"fmt"

	discogsclient "zntr.io/vynilino/internal/adapter/discogs"
	"zntr.io/vynilino/internal/domain"
)

// DiscogsSearcher is the interface the app layer depends on for Discogs searches.
type DiscogsSearcher interface {
	Search(ctx context.Context, query, searchType string) ([]domain.DiscogsResult, error)
}

// DiscogsService handles Discogs search use-cases.
type DiscogsService struct {
	client DiscogsSearcher
}

// NewDiscogsService creates a DiscogsService. client must not be nil.
func NewDiscogsService(client DiscogsSearcher) *DiscogsService {
	return &DiscogsService{client: client}
}

// Search queries Discogs and returns matching results.
// Returns user-friendly errors for rate limiting and unavailability.
func (s *DiscogsService) Search(ctx context.Context, query, searchType string) ([]domain.DiscogsResult, error) {
	results, err := s.client.Search(ctx, query, searchType)
	if err != nil {
		if errors.Is(err, discogsclient.ErrRateLimit) {
			return nil, fmt.Errorf("discogs rate limit exceeded: please try again shortly")
		}
		if errors.Is(err, discogsclient.ErrUnavailable) {
			return nil, fmt.Errorf("discogs service unavailable")
		}
		return nil, err
	}
	return results, nil
}
