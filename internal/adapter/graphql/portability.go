package graphql

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"zntr.io/vynilino/internal/app"
	"zntr.io/vynilino/internal/ctxutil"
	"zntr.io/vynilino/internal/domain"
)

const (
	maxImportBytes  = 10 << 20 // 10 MB
	importBatchSize = 50
	importBackoffMs = 10 * time.Millisecond
)

// importInProgress tracks per-user in-progress CSV imports.
// THREAT-011: keyed by userID so each user has an independent import slot.
var importInProgress sync.Map

// exportJSONHandler streams the user's collection as a JSON attachment.
func exportJSONHandler(svc *app.RecordService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := ctxutil.UserIDFromContext(r.Context())

		page, err := svc.List(r.Context(), userID, domain.RecordFilter{
			Sort:      domain.SortByCreatedAt,
			Direction: domain.SortDesc,
		}, domain.Page{Limit: 10000})
		if err != nil {
			http.Error(w, "export failed", http.StatusInternalServerError)
			return
		}

		// THREAT-013: audit log for bulk export.
		slog.InfoContext(r.Context(), "audit", "op", "export.json", "user_id", userID, "count", len(page.Records))

		date := time.Now().Format("2006-01-02")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="vynilino-export-%s.json"`, date))

		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(map[string]any{"records": page.Records})
	}
}

// csvHeaders are the canonical column names for export/import.
var csvHeaders = []string{
	"id", "title", "artist", "year", "label", "format",
	"condition", "genres", "notes", "cover_art_url", "created_at",
}

// exportCSVHandler streams the user's collection as a CSV attachment.
func exportCSVHandler(svc *app.RecordService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := ctxutil.UserIDFromContext(r.Context())

		page, err := svc.List(r.Context(), userID, domain.RecordFilter{
			Sort:      domain.SortByCreatedAt,
			Direction: domain.SortDesc,
		}, domain.Page{Limit: 10000})
		if err != nil {
			http.Error(w, "export failed", http.StatusInternalServerError)
			return
		}

		date := time.Now().Format("2006-01-02")
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="vynilino-export-%s.csv"`, date))

		// THREAT-013: audit log for bulk export.
		slog.InfoContext(r.Context(), "audit", "op", "export.csv", "user_id", userID, "count", len(page.Records))

		cw := csv.NewWriter(w)
		_ = cw.Write(csvHeaders)
		for _, rec := range page.Records {
			_ = cw.Write(recordToCSVRow(rec))
		}
		cw.Flush()
	}
}

// importCSVHandler accepts a CSV upload and creates records.
// THREAT-011: per-user import lock; concurrent imports for different users proceed independently.
func importCSVHandler(svc *app.RecordService, bus *app.EventBus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := ctxutil.UserIDFromContext(r.Context())

		// Acquire per-user slot.
		if _, loaded := importInProgress.LoadOrStore(userID, struct{}{}); loaded {
			http.Error(w, "import already in progress", http.StatusTooManyRequests)
			return
		}
		defer importInProgress.Delete(userID)

		// THREAT-013: audit log for import start.
		slog.InfoContext(r.Context(), "audit", "op", "import.csv.start", "user_id", userID)

		r.Body = http.MaxBytesReader(w, r.Body, maxImportBytes)
		if err := r.ParseMultipartForm(maxImportBytes); err != nil {
			if strings.Contains(err.Error(), "too large") {
				http.Error(w, "file too large (max 10MB)", http.StatusRequestEntityTooLarge)
			} else {
				http.Error(w, "bad request", http.StatusBadRequest)
			}
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "missing file field", http.StatusBadRequest)
			return
		}
		defer file.Close()

		if !strings.HasSuffix(strings.ToLower(header.Filename), ".csv") {
			http.Error(w, "only CSV files are accepted", http.StatusUnsupportedMediaType)
			return
		}

		imported, skipped, errs, cancelled := processCSVImport(r.Context(), svc, bus, userID, file)

		// THREAT-013: audit log for import completion.
		slog.InfoContext(r.Context(), "audit", "op", "import.csv.done",
			"user_id", userID, "imported", imported, "skipped", skipped,
			"error_count", len(errs), "cancelled", cancelled)

		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		_ = enc.Encode(map[string]any{
			"imported":  imported,
			"skipped":   skipped,
			"errors":    errs,
			"cancelled": cancelled,
		})
	}
}

func processCSVImport(ctx context.Context, svc *app.RecordService, bus *app.EventBus, userID string, r io.Reader) (imported, skipped int, errs []string, cancelled bool) {
	cr := csv.NewReader(r)
	cr.TrimLeadingSpace = true

	headers, err := cr.Read()
	if err != nil {
		return 0, 0, []string{"failed to read CSV headers: " + err.Error()}, false
	}
	isDiscogs := isDiscogsCSV(headers)
	colIdx := buildColIndex(headers, isDiscogs)

	var batch []*domain.Record

	flush := func() {
		if len(batch) == 0 {
			return
		}
		results, err := svc.CreateBatch(ctx, userID, batch)
		if err != nil {
			skipped += len(batch)
			errs = append(errs, fmt.Sprintf("batch create failed: %v", err))
		} else {
			imported += len(results)
			if bus.HasSubscribers(userID) {
				time.Sleep(importBackoffMs)
			}
		}
		batch = batch[:0]
	}

	for row := 1; ; row++ {
		select {
		case <-ctx.Done():
			flush()
			cancelled = true
			return
		default:
		}

		record, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			skipped++
			errs = append(errs, fmt.Sprintf("row %d: parse error: %v", row, err))
			continue
		}

		rec, err := rowToRecord(record, colIdx)
		if err != nil {
			skipped++
			errs = append(errs, fmt.Sprintf("row %d: %v", row, err))
			continue
		}

		batch = append(batch, rec)
		if len(batch) >= importBatchSize {
			flush()
		}
	}
	flush()
	return
}

// isDiscogsCSV detects a Discogs-format CSV by header names.
func isDiscogsCSV(headers []string) bool {
	for _, h := range headers {
		if h == "Catalog#" || h == "release_id" {
			return true
		}
	}
	return false
}

// discogsMapping maps Discogs column names to vynilino field names.
var discogsMapping = map[string]string{
	"Title":    "title",
	"Artist":   "artist",
	"Label":    "label",
	"Format":   "format",
	"Released": "year",
	"Notes":    "notes",
}

func buildColIndex(headers []string, discogs bool) map[string]int {
	idx := make(map[string]int, len(headers))
	for i, h := range headers {
		key := h
		if discogs {
			if mapped, ok := discogsMapping[h]; ok {
				key = mapped
			}
		}
		idx[strings.ToLower(strings.TrimSpace(key))] = i
	}
	return idx
}

func rowToRecord(row []string, colIdx map[string]int) (*domain.Record, error) {
	get := func(name string) string {
		if i, ok := colIdx[name]; ok && i < len(row) {
			return strings.TrimSpace(row[i])
		}
		return ""
	}

	title := get("title")
	artist := get("artist")
	if title == "" || artist == "" {
		return nil, fmt.Errorf("title and artist are required")
	}

	rec := &domain.Record{Title: title, Artist: artist}

	if y := get("year"); y != "" {
		n, err := strconv.Atoi(y)
		if err == nil {
			rec.Year = &n
		}
	}
	if l := get("label"); l != "" {
		rec.Label = &l
	}
	if f := get("format"); f != "" {
		format := domain.Format(f)
		rec.Format = &format
	}
	if c := get("condition"); c != "" {
		cond := domain.Condition(c)
		rec.Condition = &cond
	}
	if g := get("genres"); g != "" {
		rec.Genres = strings.Split(g, "|")
	}
	if n := get("notes"); n != "" {
		rec.Notes = &n
	}
	return rec, nil
}

// sanitizeCSVField prevents CSV formula injection (THREAT-012).
// Leading characters that trigger formula execution in spreadsheet software
// are neutralised by prepending a tab character.
func sanitizeCSVField(s string) string {
	if len(s) == 0 {
		return s
	}
	switch s[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "\t" + s
	}
	return s
}

func recordToCSVRow(rec *domain.Record) []string {
	year := ""
	if rec.Year != nil {
		year = strconv.Itoa(*rec.Year)
	}
	label := ""
	if rec.Label != nil {
		label = sanitizeCSVField(*rec.Label)
	}
	format := ""
	if rec.Format != nil {
		format = string(*rec.Format)
	}
	condition := ""
	if rec.Condition != nil {
		condition = string(*rec.Condition)
	}
	genres := sanitizeCSVField(strings.Join(rec.Genres, "|"))
	notes := ""
	if rec.Notes != nil {
		notes = sanitizeCSVField(*rec.Notes)
	}
	coverArt := ""
	if rec.CoverArtURL != nil {
		coverArt = *rec.CoverArtURL
	}
	return []string{
		rec.ID, sanitizeCSVField(rec.Title), sanitizeCSVField(rec.Artist),
		year, label, format, condition, genres, notes, coverArt,
		rec.CreatedAt.Format(time.RFC3339),
	}
}
