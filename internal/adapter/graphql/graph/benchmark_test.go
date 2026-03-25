package graph

import (
	"context"
	"testing"
	"time"

	"zntr.io/vynilino/internal/app"
	"zntr.io/vynilino/internal/ctxutil"
	"zntr.io/vynilino/internal/domain"
)

// stubRecordRepo implements domain.RecordRepository with an in-memory dataset.
// Only List is exercised by the Records resolver; all other methods are no-ops.
type stubRecordRepo struct {
	page *domain.RecordPage
}

func (s *stubRecordRepo) Create(_ context.Context, r *domain.Record) (*domain.Record, error) {
	return r, nil
}
func (s *stubRecordRepo) CreateBatch(_ context.Context, records []*domain.Record) ([]*domain.Record, error) {
	return records, nil
}
func (s *stubRecordRepo) GetByID(_ context.Context, _, _ string) (*domain.Record, error) {
	return nil, nil
}
func (s *stubRecordRepo) Update(_ context.Context, r *domain.Record) (*domain.Record, error) {
	return r, nil
}
func (s *stubRecordRepo) Delete(_ context.Context, _, _ string) error { return nil }
func (s *stubRecordRepo) List(_ context.Context, _ string, _ domain.RecordFilter, _ domain.Page) (*domain.RecordPage, error) {
	return s.page, nil
}
func (s *stubRecordRepo) FindDuplicate(_ context.Context, _, _, _ string) (string, error) {
	return "", nil
}
func (s *stubRecordRepo) UpdateCoverArt(_ context.Context, _, _, _ string) error { return nil }

// buildRecordPage returns n fully-populated records, simulating the worst-case
// result set for a maximally nested Records query (all 14 scalar fields set).
func buildRecordPage(n int) *domain.RecordPage {
	records := make([]*domain.Record, n)
	for i := range records {
		year := 1970 + i%55
		label := "Label"
		format := domain.FormatLP
		cond := domain.ConditionNearMint
		notes := "notes"
		coverArtURL := "/media/cover-art/user/record.jpg"
		discogsID := "12345"
		fav := false
		personalNote := "personal"
		records[i] = &domain.Record{
			ID:           "id",
			UserID:       "bench-user",
			Title:        "Title",
			Artist:       "Artist",
			Year:         &year,
			Label:        &label,
			Format:       &format,
			Condition:    &cond,
			Genres:       []string{"Rock", "Jazz"},
			Notes:        &notes,
			CoverArtURL:  &coverArtURL,
			DiscogsID:    &discogsID,
			Favorite:     &fav,
			PersonalNote: &personalNote,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
	}
	return &domain.RecordPage{Records: records, TotalCount: n}
}

// BenchmarkRecordsQueryMaxDepth measures resolver throughput at the maximum
// page size (first:100) with all 14 scalar fields populated — the worst-case
// depth-3 query (records → edges → node + pageInfo + totalCount).
// Use these ns/op figures to calibrate FixedComplexityLimit in server.go.
func BenchmarkRecordsQueryMaxDepth(b *testing.B) {
	repo := &stubRecordRepo{page: buildRecordPage(100)}
	svc := app.NewRecordService(repo, app.NewEventBus())
	r := &Resolver{RecordSvc: svc}

	first := 100
	ctx := ctxutil.WithUserID(context.Background(), "bench-user")

	b.ResetTimer()
	for b.Loop() {
		if _, err := r.Query().Records(ctx, &first, nil, nil, nil); err != nil {
			b.Fatal(err)
		}
	}
}
