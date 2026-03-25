package graphql

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"zntr.io/vynilino/internal/app"
	"zntr.io/vynilino/internal/ctxutil"
	"zntr.io/vynilino/internal/domain"
)

const maxImportBytes = 10 << 20 // 10 MB

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

		cw := csv.NewWriter(w)
		_ = cw.Write(csvHeaders)
		for _, rec := range page.Records {
			_ = cw.Write(recordToCSVRow(rec))
		}
		cw.Flush()
	}
}

// importCSVHandler accepts a CSV upload and creates records.
func importCSVHandler(svc *app.RecordService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := ctxutil.UserIDFromContext(r.Context())

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

		imported, skipped, errs := processCSVImport(r.Context(), svc, userID, file)

		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		_ = enc.Encode(map[string]any{
			"imported": imported,
			"skipped":  skipped,
			"errors":   errs,
		})
	}
}

func processCSVImport(ctx context.Context, svc *app.RecordService, userID string, r io.Reader) (imported, skipped int, errs []string) {
	cr := csv.NewReader(r)
	cr.TrimLeadingSpace = true

	headers, err := cr.Read()
	if err != nil {
		return 0, 0, []string{"failed to read CSV headers: " + err.Error()}
	}
	isDiscogs := isDiscogsCSV(headers)
	colIdx := buildColIndex(headers, isDiscogs)

	for row := 1; ; row++ {
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

		if _, err := svc.Create(ctx, userID, rec); err != nil {
			skipped++
			errs = append(errs, fmt.Sprintf("row %d: create failed: %v", row, err))
			continue
		}
		imported++
	}
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

func recordToCSVRow(rec *domain.Record) []string {
	year := ""
	if rec.Year != nil {
		year = strconv.Itoa(*rec.Year)
	}
	label := ""
	if rec.Label != nil {
		label = *rec.Label
	}
	format := ""
	if rec.Format != nil {
		format = string(*rec.Format)
	}
	condition := ""
	if rec.Condition != nil {
		condition = string(*rec.Condition)
	}
	genres := strings.Join(rec.Genres, "|")
	notes := ""
	if rec.Notes != nil {
		notes = *rec.Notes
	}
	coverArt := ""
	if rec.CoverArtURL != nil {
		coverArt = *rec.CoverArtURL
	}
	return []string{
		rec.ID, rec.Title, rec.Artist, year, label, format,
		condition, genres, notes, coverArt,
		rec.CreatedAt.Format(time.RFC3339),
	}
}

