package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"zntr.io/vynilino/internal/adapter/storage/sqlite/sqlcdb"
	"zntr.io/vynilino/internal/domain"
)

type recordRepository struct {
	db *sql.DB
	q  *sqlcdb.Queries
}

// NewRecordRepository returns a domain.RecordRepository backed by SQLite.
func NewRecordRepository(db *sql.DB) domain.RecordRepository {
	return &recordRepository{db: db, q: sqlcdb.New(db)}
}

func (r *recordRepository) Create(ctx context.Context, rec *domain.Record) (*domain.Record, error) {
	now := time.Now()
	rec.ID = uuid.NewString()
	rec.CreatedAt = now
	rec.UpdatedAt = now

	start := time.Now()
	row, err := r.q.CreateRecord(ctx, sqlcdb.CreateRecordParams{
		ID:           rec.ID,
		UserID:       rec.UserID,
		Title:        rec.Title,
		Artist:       rec.Artist,
		Year:         toNullInt64(rec.Year),
		Label:        toNullString(rec.Label),
		Format:       toNullStringV((*string)(rec.Format)),
		Condition:    toNullStringV((*string)(rec.Condition)),
		Genre:        sql.NullString{String: jsonStrings(rec.Genres), Valid: true},
		Notes:        toNullString(rec.Notes),
		CoverArtUrl:  toNullString(rec.CoverArtURL),
		DiscogsID:    toNullString(rec.DiscogsID),
		Favorite:     boolPtrToInt(rec.Favorite),
		PersonalNote: toNullString(rec.PersonalNote),
		CreatedAt:    now.Unix(),
		UpdatedAt:    now.Unix(),
	})
	logWriteDuration(ctx, "record.Create", time.Since(start))
	if err != nil {
		return nil, err
	}
	return recordFromRow(row), nil
}

// CreateBatch inserts a slice of records within a single SQLite transaction,
// reducing write-lock hold time compared to one transaction per row.
func (r *recordRepository) CreateBatch(ctx context.Context, records []*domain.Record) ([]*domain.Record, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	qtx := sqlcdb.New(tx)
	now := time.Now()
	start := now

	result := make([]*domain.Record, 0, len(records))
	for _, rec := range records {
		rec.ID = uuid.NewString()
		rec.CreatedAt = now
		rec.UpdatedAt = now

		row, err := qtx.CreateRecord(ctx, sqlcdb.CreateRecordParams{
			ID:           rec.ID,
			UserID:       rec.UserID,
			Title:        rec.Title,
			Artist:       rec.Artist,
			Year:         toNullInt64(rec.Year),
			Label:        toNullString(rec.Label),
			Format:       toNullStringV((*string)(rec.Format)),
			Condition:    toNullStringV((*string)(rec.Condition)),
			Genre:        sql.NullString{String: jsonStrings(rec.Genres), Valid: true},
			Notes:        toNullString(rec.Notes),
			CoverArtUrl:  toNullString(rec.CoverArtURL),
			DiscogsID:    toNullString(rec.DiscogsID),
			Favorite:     boolPtrToInt(rec.Favorite),
			PersonalNote: toNullString(rec.PersonalNote),
			CreatedAt:    now.Unix(),
			UpdatedAt:    now.Unix(),
		})
		if err != nil {
			return nil, err
		}
		result = append(result, recordFromRow(row))
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	logWriteDuration(ctx, "record.CreateBatch", time.Since(start))
	return result, nil
}

// logWriteDuration emits a WARN log when a write operation exceeds 1 second.
func logWriteDuration(ctx context.Context, op string, d time.Duration) {
	ms := d.Milliseconds()
	if ms > 1000 {
		slog.WarnContext(ctx, "db write slow",
			"op", op,
			"db_write_duration_ms", ms,
			"busy_timeout_approached", ms > 3000,
		)
	}
}

func (r *recordRepository) GetByID(ctx context.Context, id, userID string) (*domain.Record, error) {
	row, err := r.q.GetRecordByID(ctx, sqlcdb.GetRecordByIDParams{ID: id, UserID: userID})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return recordFromRow(row), nil
}

func (r *recordRepository) Update(ctx context.Context, rec *domain.Record) (*domain.Record, error) {
	now := time.Now()
	row, err := r.q.UpdateRecord(ctx, sqlcdb.UpdateRecordParams{
		Title:        rec.Title,
		Artist:       rec.Artist,
		Year:         toNullInt64(rec.Year),
		Label:        toNullString(rec.Label),
		Format:       toNullStringV((*string)(rec.Format)),
		Condition:    toNullStringV((*string)(rec.Condition)),
		Genre:        sql.NullString{String: jsonStrings(rec.Genres), Valid: true},
		Notes:        toNullString(rec.Notes),
		CoverArtUrl:  toNullString(rec.CoverArtURL),
		Favorite:     boolPtrToInt(rec.Favorite),
		PersonalNote: toNullString(rec.PersonalNote),
		UpdatedAt:    now.Unix(),
		ID:           rec.ID,
		UserID:       rec.UserID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return recordFromRow(row), nil
}

func (r *recordRepository) Delete(ctx context.Context, id, userID string) error {
	return r.q.DeleteRecord(ctx, sqlcdb.DeleteRecordParams{ID: id, UserID: userID})
}

func (r *recordRepository) List(ctx context.Context, userID string, filter domain.RecordFilter, page domain.Page) (*domain.RecordPage, error) {
	rows, total, err := r.listFiltered(ctx, userID, filter, page)
	if err != nil {
		return nil, err
	}
	records := make([]*domain.Record, len(rows))
	for i, row := range rows {
		records[i] = recordFromRow(row)
	}
	return &domain.RecordPage{Records: records, TotalCount: total}, nil
}

func (r *recordRepository) listFiltered(ctx context.Context, userID string, filter domain.RecordFilter, page domain.Page) ([]sqlcdb.Record, int, error) {
	var where []string
	var args []any

	where = append(where, "user_id = ?")
	args = append(args, userID)

	if filter.Search != nil && *filter.Search != "" {
		where = append(where, "id IN (SELECT rowid FROM records_fts WHERE records_fts MATCH ?)")
		args = append(args, *filter.Search+"*")
	}
	if filter.Artist != nil {
		where = append(where, "LOWER(artist) LIKE LOWER(?)")
		args = append(args, "%"+*filter.Artist+"%")
	}
	if filter.Genre != nil {
		where = append(where, "genre LIKE ?")
		args = append(args, "%"+*filter.Genre+"%")
	}
	if filter.YearMin != nil {
		where = append(where, "year >= ?")
		args = append(args, *filter.YearMin)
	}
	if filter.YearMax != nil {
		where = append(where, "year <= ?")
		args = append(args, *filter.YearMax)
	}
	if filter.Format != nil {
		where = append(where, "format = ?")
		args = append(args, string(*filter.Format))
	}
	if filter.ConditionMin != nil {
		grades := conditionGrades(*filter.ConditionMin)
		placeholders := make([]string, len(grades))
		for i, g := range grades {
			placeholders[i] = "?"
			args = append(args, g)
		}
		where = append(where, "condition IN ("+strings.Join(placeholders, ",")+")")
	}
	if filter.FavoritesOnly {
		where = append(where, "favorite = 1")
	}

	sortCol := "created_at"
	switch filter.Sort {
	case domain.SortByTitle:
		sortCol = "title"
	case domain.SortByArtist:
		sortCol = "artist"
	case domain.SortByYear:
		sortCol = "year"
	case domain.SortByUpdatedAt:
		sortCol = "updated_at"
	}
	dir := "DESC"
	if filter.Direction == domain.SortAsc {
		dir = "ASC"
	}

	whereClause := strings.Join(where, " AND ")
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM records WHERE %s", whereClause)
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := page.Limit
	if limit <= 0 {
		limit = 20
	}
	dataQuery := fmt.Sprintf(
		"SELECT * FROM records WHERE %s ORDER BY %s %s NULLS LAST LIMIT ? OFFSET ?",
		whereClause, sortCol, dir,
	)
	dataArgs := make([]any, len(args), len(args)+2)
	copy(dataArgs, args)
	dataArgs = append(dataArgs, limit, page.Offset)
	rows, err := r.db.QueryContext(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var records []sqlcdb.Record
	for rows.Next() {
		var rec sqlcdb.Record
		if err := rows.Scan(
			&rec.ID, &rec.UserID, &rec.Title, &rec.Artist,
			&rec.Year, &rec.Label, &rec.Format, &rec.Condition,
			&rec.Genre, &rec.Notes, &rec.CoverArtUrl,
			&rec.CreatedAt, &rec.UpdatedAt, &rec.DiscogsID,
			&rec.Favorite, &rec.PersonalNote,
		); err != nil {
			return nil, 0, err
		}
		records = append(records, rec)
	}
	return records, total, rows.Err()
}

func (r *recordRepository) FindDuplicate(ctx context.Context, userID, title, artist string) (string, error) {
	id, err := r.q.FindDuplicateRecord(ctx, sqlcdb.FindDuplicateRecordParams{
		UserID:  userID,
		LOWER:   title,
		LOWER_2: artist,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return id, nil
}

func (r *recordRepository) UpdateCoverArt(ctx context.Context, id, userID, url string) error {
	return r.q.UpdateRecordCoverArt(ctx, sqlcdb.UpdateRecordCoverArtParams{
		CoverArtUrl: sql.NullString{String: url, Valid: true},
		UpdatedAt:   time.Now().Unix(),
		ID:          id,
		UserID:      userID,
	})
}

// --- helpers ---

func recordFromRow(row sqlcdb.Record) *domain.Record {
	fav := row.Favorite != 0
	rec := &domain.Record{
		ID:        row.ID,
		UserID:    row.UserID,
		Title:     row.Title,
		Artist:    row.Artist,
		Genres:    parseJSONStrings(row.Genre.String),
		Favorite:  &fav,
		CreatedAt: time.Unix(row.CreatedAt, 0),
		UpdatedAt: time.Unix(row.UpdatedAt, 0),
	}
	if row.Year.Valid {
		y := int(row.Year.Int64)
		rec.Year = &y
	}
	if row.Label.Valid {
		rec.Label = &row.Label.String
	}
	if row.Format.Valid {
		f := domain.Format(row.Format.String)
		rec.Format = &f
	}
	if row.Condition.Valid {
		c := domain.Condition(row.Condition.String)
		rec.Condition = &c
	}
	if row.Notes.Valid {
		rec.Notes = &row.Notes.String
	}
	if row.CoverArtUrl.Valid {
		rec.CoverArtURL = &row.CoverArtUrl.String
	}
	if row.DiscogsID.Valid {
		rec.DiscogsID = &row.DiscogsID.String
	}
	if row.PersonalNote.Valid {
		rec.PersonalNote = &row.PersonalNote.String
	}
	return rec
}

func boolPtrToInt(b *bool) int64 {
	if b != nil && *b {
		return 1
	}
	return 0
}

func toNullInt64(v *int) sql.NullInt64 {
	if v == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*v), Valid: true}
}

func toNullString(v *string) sql.NullString {
	if v == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *v, Valid: true}
}

func toNullStringV(v *string) sql.NullString {
	return toNullString(v)
}

// conditionGrades returns all conditions >= minCondition.
func conditionGrades(minCond domain.Condition) []string {
	minRank := domain.ConditionOrder[minCond]
	var out []string
	for c, rank := range domain.ConditionOrder {
		if rank >= minRank {
			out = append(out, string(c))
		}
	}
	return out
}
