package app

import (
	"context"
	"fmt"
	"log/slog"

	"zntr.io/vynilino/internal/domain"
)

// RecordChangedEvent is published on the subscription bus when a record mutates.
type RecordChangedEvent struct {
	Type     string // "CREATED" | "UPDATED" | "DELETED"
	Record   *domain.Record
	RecordID string
}

// RecordService handles collection management business logic.
type RecordService struct {
	records domain.RecordRepository
	bus     *EventBus
}

// NewRecordService constructs a RecordService.
func NewRecordService(records domain.RecordRepository, bus *EventBus) *RecordService {
	return &RecordService{records: records, bus: bus}
}

// CreateResult wraps a created record plus an optional duplicate warning.
type CreateResult struct {
	Record           *domain.Record
	DuplicateWarning string
}

// Create adds a new record to the collection.
func (s *RecordService) Create(ctx context.Context, userID string, r *domain.Record) (*CreateResult, error) {
	r.UserID = userID

	var warning string
	dupID, err := s.records.FindDuplicate(ctx, userID, r.Title, r.Artist)
	if err != nil {
		return nil, err
	}
	if dupID != "" {
		warning = fmt.Sprintf("A record with the same title and artist already exists (id: %s)", dupID)
	}

	created, err := s.records.Create(ctx, r)
	if err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "audit", "op", "record.create", "user_id", userID, "record_id", created.ID)
	s.bus.Publish(ctx, userID, RecordChangedEvent{Type: "CREATED", Record: created, RecordID: created.ID})
	return &CreateResult{Record: created, DuplicateWarning: warning}, nil
}

// GetByID fetches a record owned by the user.
func (s *RecordService) GetByID(ctx context.Context, id, userID string) (*domain.Record, error) {
	return s.records.GetByID(ctx, id, userID)
}

// Update modifies an existing record.
func (s *RecordService) Update(ctx context.Context, userID string, r *domain.Record) (*domain.Record, error) {
	existing, err := s.records.GetByID(ctx, r.ID, userID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("record %s not found", r.ID)
	}

	// Merge: only overwrite non-nil fields from the update input.
	applyUpdates(existing, r)

	updated, err := s.records.Update(ctx, existing)
	if err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "audit", "op", "record.update", "user_id", userID, "record_id", updated.ID)
	s.bus.Publish(ctx, userID, RecordChangedEvent{Type: "UPDATED", Record: updated, RecordID: updated.ID})
	return updated, nil
}

// Delete removes a record from the collection.
func (s *RecordService) Delete(ctx context.Context, id, userID string) error {
	if err := s.records.Delete(ctx, id, userID); err != nil {
		return err
	}
	slog.InfoContext(ctx, "audit", "op", "record.delete", "user_id", userID, "record_id", id)
	s.bus.Publish(ctx, userID, RecordChangedEvent{Type: "DELETED", RecordID: id})
	return nil
}

// CreateBatch adds multiple records in a single database transaction.
// Duplicate detection is skipped for bulk import performance.
func (s *RecordService) CreateBatch(ctx context.Context, userID string, records []*domain.Record) ([]*CreateResult, error) {
	for _, r := range records {
		r.UserID = userID
	}
	created, err := s.records.CreateBatch(ctx, records)
	if err != nil {
		return nil, err
	}
	results := make([]*CreateResult, len(created))
	for i, rec := range created {
		s.bus.Publish(ctx, userID, RecordChangedEvent{Type: "CREATED", Record: rec, RecordID: rec.ID})
		results[i] = &CreateResult{Record: rec}
	}
	return results, nil
}

// List returns a paginated, filtered collection.
func (s *RecordService) List(ctx context.Context, userID string, filter domain.RecordFilter, page domain.Page) (*domain.RecordPage, error) {
	return s.records.List(ctx, userID, filter, page)
}

// Subscribe registers a subscription channel for the given user.
func (s *RecordService) Subscribe(userID string) (<-chan RecordChangedEvent, func()) {
	return s.bus.Subscribe(userID)
}

// applyUpdates merges non-nil fields from src into dst.
func applyUpdates(dst, src *domain.Record) {
	if src.Title != "" {
		dst.Title = src.Title
	}
	if src.Artist != "" {
		dst.Artist = src.Artist
	}
	if src.Year != nil {
		dst.Year = src.Year
	}
	if src.Label != nil {
		dst.Label = src.Label
	}
	if src.Format != nil {
		dst.Format = src.Format
	}
	if src.Condition != nil {
		dst.Condition = src.Condition
	}
	if src.Genres != nil {
		dst.Genres = src.Genres
	}
	if src.Notes != nil {
		dst.Notes = src.Notes
	}
	if src.CoverArtURL != nil {
		dst.CoverArtURL = src.CoverArtURL
	}
	if src.Favorite != nil {
		dst.Favorite = src.Favorite
	}
	if src.PersonalNote != nil {
		dst.PersonalNote = src.PersonalNote
	}
}
