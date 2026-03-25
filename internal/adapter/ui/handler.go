// Package ui serves the embedded SPA and exposes thin REST helpers (/api/me, /api/upload).
package ui

import (
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"strings"

	"zntr.io/vynilino/internal/adapter/filestore"
	"zntr.io/vynilino/internal/ctxutil"
	"zntr.io/vynilino/internal/domain"
)

// Handler serves the embedded SPA and the /api/me + /api/upload endpoints.
type Handler struct {
	static   fs.FS
	userRepo domain.UserRepository
	fs       *filestore.FileStore
}

// New creates a Handler. staticFiles should be the embedded ui/dist FS
// (or a sub-FS of it).
func New(staticFiles fs.FS, userRepo domain.UserRepository, fs *filestore.FileStore) *Handler {
	return &Handler{
		static:   staticFiles,
		userRepo: userRepo,
		fs:       fs,
	}
}

// MeHandler handles GET /api/me.
func (h *Handler) MeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := ctxutil.UserIDFromContext(r.Context())
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthenticated"}`))
			return
		}

		user, err := h.userRepo.GetByID(r.Context(), userID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthenticated"}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"id":    user.ID,
			"email": user.Email,
		})
	}
}

// UploadHandler handles POST /api/upload (cover art, ≤ 5 MB).
func (h *Handler) UploadHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := ctxutil.UserIDFromContext(r.Context())
		if !ok {
			http.Error(w, `{"error":"unauthenticated"}`, http.StatusUnauthorized)
			return
		}

		if err := r.ParseMultipartForm(filestore.MaxCoverArtBytes + 1); err != nil {
			http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, `{"error":"missing file field"}`, http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Use a temporary record ID placeholder for upload-before-create flow.
		recordID := r.FormValue("recordId")
		if recordID == "" {
			recordID = "tmp"
		}

		url, err := h.fs.StoreCoverArt(userID, recordID, file)
		if err != nil {
			switch err {
			case filestore.ErrFileTooLarge:
				http.Error(w, `{"error":"file too large"}`, http.StatusRequestEntityTooLarge)
			case filestore.ErrUnsupportedMediaType:
				http.Error(w, `{"error":"unsupported media type"}`, http.StatusUnsupportedMediaType)
			default:
				http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"url":"` + url + `"}`))
	}
}

// SPAHandler returns an http.Handler that serves static files from the embedded
// FS and falls back to index.html for unknown paths (client-side routing).
//
// The /login path is served from the standalone login.html bundle.
// API routes (/graphql, /api/, /auth/, /media/, /export/, /import/, /health)
// must be registered BEFORE this handler in the router.
func (h *Handler) SPAHandler() http.Handler {
	fileServer := http.FileServer(http.FS(h.static))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only serve GET/HEAD for the SPA.
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}

		path := r.URL.Path

		// Serve the standalone login bundle for /login (without extension).
		if path == "/login" {
			loginHTML, err := h.static.Open("login.html")
			if err != nil {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			defer loginHTML.Close()
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = io.Copy(w, loginHTML)
			return
		}

		// Check if the path exists as a real file in the embedded FS.
		if _, err := fs.Stat(h.static, strings.TrimPrefix(path, "/")); err == nil {
			// Set immutable cache headers for hashed asset files.
			if strings.HasPrefix(path, "/assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			fileServer.ServeHTTP(w, r)
			return
		}

		// Fall back to index.html for all other SPA routes.
		index, err := h.static.Open("index.html")
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		defer index.Close()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.Copy(w, index)
	})
}
