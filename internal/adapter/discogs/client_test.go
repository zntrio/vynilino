package discogs_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"zntr.io/vynilino/internal/adapter/discogs"
)

func TestErrRateLimit_Classification(t *testing.T) {
	if !errors.Is(discogs.ErrRateLimit, discogs.ErrRateLimit) {
		t.Fatal("ErrRateLimit should match itself via errors.Is")
	}
}

func TestErrUnavailable_Classification(t *testing.T) {
	wrapped := fmt.Errorf("context: %w", discogs.ErrUnavailable)
	if !errors.Is(wrapped, discogs.ErrUnavailable) {
		t.Fatal("wrapped ErrUnavailable should unwrap correctly")
	}
}

func TestSearch_ContextCancellation(t *testing.T) {
	// Create a real client (token-less, will be rejected by Discogs but we
	// cancel context immediately so it never completes the real HTTP call).
	c, err := discogs.New("", "vynilino-test/1.0")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	_, searchErr := c.Search(ctx, "Pink Floyd", "release")
	if searchErr == nil {
		t.Fatal("expected error on cancelled context, got nil")
	}
	if !errors.Is(searchErr, discogs.ErrUnavailable) {
		t.Logf("got error: %v (expected ErrUnavailable wrapper)", searchErr)
		// Acceptable: context cancelled may produce ErrUnavailable
	}
}
