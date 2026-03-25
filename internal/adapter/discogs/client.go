package discogs

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	godiscogs "github.com/irlndts/go-discogs"

	"zntr.io/vynilino/internal/domain"
)

// ErrRateLimit is returned when the Discogs API responds with HTTP 429.
var ErrRateLimit = errors.New("discogs rate limit exceeded")

// ErrUnavailable is returned when the Discogs API cannot be reached or times out.
var ErrUnavailable = errors.New("discogs service unavailable")

// Client wraps the go-discogs library to expose context-aware search.
type Client struct {
	discogs godiscogs.Discogs
}

// New creates a Discogs client. token may be empty for unauthenticated access
// (lower rate limit). userAgent should identify the application.
func New(token, userAgent string) (*Client, error) {
	d, err := godiscogs.New(&godiscogs.Options{
		UserAgent: userAgent,
		Token:     token,
	})
	if err != nil {
		return nil, fmt.Errorf("discogs client init: %w", err)
	}
	return &Client{discogs: d}, nil
}

// Search queries the Discogs database. searchType should be one of "release",
// "master", or "" (all). Returns ErrRateLimit or ErrUnavailable on API errors.
func (c *Client) Search(ctx context.Context, query, searchType string) ([]domain.DiscogsResult, error) {
	type result struct {
		search *godiscogs.Search
		err    error
	}

	ch := make(chan result, 1)
	go func() {
		s, err := c.discogs.Search(godiscogs.SearchRequest{
			Q:       query,
			Type:    searchType,
			PerPage: 25,
			Page:    1,
		})
		ch <- result{s, err}
	}()

	select {
	case <-ctx.Done():
		return nil, ErrUnavailable
	case r := <-ch:
		if r.err != nil {
			errStr := r.err.Error()
			if strings.Contains(errStr, "429") {
				return nil, ErrRateLimit
			}
			return nil, fmt.Errorf("%w: %s", ErrUnavailable, errStr)
		}
		return toDiscogsResults(r.search.Results), nil
	}
}

// toDiscogsResults maps go-discogs Results to domain.DiscogsResult slice.
func toDiscogsResults(results []godiscogs.Result) []domain.DiscogsResult {
	out := make([]domain.DiscogsResult, 0, len(results))
	for _, r := range results {
		dr := domain.DiscogsResult{
			DiscogsID: strconv.Itoa(r.ID),
			Title:     r.Title,
		}
		if r.Year != "" {
			if y, err := strconv.Atoi(r.Year); err == nil {
				dr.Year = &y
			}
		}
		if r.Country != "" {
			dr.Country = &r.Country
		}
		if r.Thumb != "" {
			dr.ThumbURL = &r.Thumb
		}
		if len(r.Label) > 0 {
			l := strings.Join(r.Label, ", ")
			dr.Label = &l
		}
		if len(r.Format) > 0 {
			f := strings.Join(r.Format, ", ")
			dr.Format = &f
		}
		// Discogs title for releases is usually "Artist - Album", split on " - "
		if idx := strings.Index(r.Title, " - "); idx != -1 {
			artist := r.Title[:idx]
			dr.Artist = &artist
			titleOnly := r.Title[idx+3:]
			dr.Title = titleOnly
		}
		out = append(out, dr)
	}
	return out
}
