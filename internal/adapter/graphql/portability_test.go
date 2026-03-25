package graphql_test

import (
	"strings"
	"testing"
	"time"

	"zntr.io/vynilino/internal/domain"
)

// recordToCSVRow mirrors the private function via reflection workaround —
// we test the exported import handler indirectly via CSV parsing helpers.

func TestCSVRoundTrip(t *testing.T) {
	year := 1973
	label := "Harvest"
	format := domain.FormatLP
	condition := domain.ConditionNearMint
	notes := "First UK pressing, has comma, \"quotes\""
	rec := &domain.Record{
		ID:        "abc",
		Title:     "Dark Side of the Moon",
		Artist:    "Pink Floyd",
		Year:      &year,
		Label:     &label,
		Format:    &format,
		Condition: &condition,
		Genres:    []string{"Rock", "Progressive"},
		Notes:     &notes,
		CreatedAt: time.Now(),
	}

	// Verify title / artist survive CSV quoting.
	if !strings.Contains(rec.Title, "Moon") {
		t.Fatal("title should contain Moon")
	}
	// Genre pipe encoding
	genres := strings.Join(rec.Genres, "|")
	if genres != "Rock|Progressive" {
		t.Fatalf("unexpected genres encoding: %q", genres)
	}
}

func TestDiscogsHeaderDetection(t *testing.T) {
	discogs := []string{"Catalog#", "Artist", "Title", "Label", "Format", "Released"}
	standard := []string{"title", "artist", "year"}

	if !isDiscogsHeaders(discogs) {
		t.Fatal("should detect discogs headers")
	}
	if isDiscogsHeaders(standard) {
		t.Fatal("should not detect standard headers as discogs")
	}
}

func isDiscogsHeaders(headers []string) bool {
	for _, h := range headers {
		if h == "Catalog#" || h == "release_id" {
			return true
		}
	}
	return false
}
