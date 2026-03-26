package domain

import (
	"context"
	"time"
)

// Format represents vinyl record physical formats.
type Format string

const (
	FormatLP     Format = "LP"
	FormatEP     Format = "EP"
	FormatSingle Format = "Single"
	Format7Inch  Format = `7"`
	Format10Inch Format = `10"`
	Format12Inch Format = `12"`
)

// Condition represents the Goldmine grading scale.
type Condition string

const (
	ConditionMint        Condition = "Mint"
	ConditionNearMint    Condition = "Near Mint"
	ConditionVeryGoodPls Condition = "Very Good Plus"
	ConditionVeryGood    Condition = "Very Good"
	ConditionGoodPls     Condition = "Good Plus"
	ConditionGood        Condition = "Good"
	ConditionFair        Condition = "Fair"
	ConditionPoor        Condition = "Poor"
)

// ConditionOrder maps conditions to a sortable integer (higher = better).
var ConditionOrder = map[Condition]int{
	ConditionMint:        8,
	ConditionNearMint:    7,
	ConditionVeryGoodPls: 6,
	ConditionVeryGood:    5,
	ConditionGoodPls:     4,
	ConditionGood:        3,
	ConditionFair:        2,
	ConditionPoor:        1,
}

// Record represents a vinyl record in a user's collection.
type Record struct {
	ID           string
	UserID       string
	Title        string
	Artist       string
	Year         *int
	Label        *string
	Format       *Format
	Condition    *Condition
	Genres       []string
	Notes        *string
	CoverArtURL  *string
	DiscogsID    *string
	Favorite     *bool
	PersonalNote *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// SortField identifies a field by which records can be sorted.
type SortField string

const (
	SortByTitle     SortField = "title"
	SortByArtist    SortField = "artist"
	SortByYear      SortField = "year"
	SortByCreatedAt SortField = "created_at"
	SortByUpdatedAt SortField = "updated_at"
)

// SortDirection is ASC or DESC.
type SortDirection string

const (
	SortAsc  SortDirection = "ASC"
	SortDesc SortDirection = "DESC"
)

// RecordFilter contains search/filter criteria for listing records.
type RecordFilter struct {
	Search        *string
	Artist        *string
	Genre         *string
	YearMin       *int
	YearMax       *int
	Format        *Format
	ConditionMin  *Condition
	FavoritesOnly bool
	Sort          SortField
	Direction     SortDirection
}

// Page contains pagination parameters.
type Page struct {
	Limit  int
	Offset int
}

// RecordPage is a paginated list of records.
type RecordPage struct {
	Records    []*Record
	TotalCount int
}

// DiscogsResult holds metadata returned from a Discogs database search.
type DiscogsResult struct {
	DiscogsID string
	Title     string
	Artist    *string
	Year      *int
	Label     *string
	Format    *string
	ThumbURL  *string
	Country   *string
}

// RecordRepository defines persistence operations for records.
type RecordRepository interface {
	Create(ctx context.Context, r *Record) (*Record, error)
	CreateBatch(ctx context.Context, records []*Record) ([]*Record, error)
	GetByID(ctx context.Context, id, userID string) (*Record, error)
	Update(ctx context.Context, r *Record) (*Record, error)
	Delete(ctx context.Context, id, userID string) error
	List(ctx context.Context, userID string, filter RecordFilter, page Page) (*RecordPage, error)
	FindDuplicate(ctx context.Context, userID, title, artist string) (string, error) // returns ID or ""
	UpdateCoverArt(ctx context.Context, id, userID, url string) error
}
