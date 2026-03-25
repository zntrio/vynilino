package ui_test

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"zntr.io/vynilino/internal/adapter/ui"
)

// stubFS builds a minimal in-memory FS with index.html, login.html, and a hashed asset.
func stubFS() fs.FS {
	return fstest.MapFS{
		"index.html":            &fstest.MapFile{Data: []byte("<html>app</html>")},
		"login.html":            &fstest.MapFile{Data: []byte("<html>login</html>")},
		"assets/main-abc123.js": &fstest.MapFile{Data: []byte("console.log('hi')")},
	}
}

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	h := ui.New(stubFS(), nil, nil)
	return h.SPAHandler()
}

func TestSPAHandler_RootReturnsIndexHTML(t *testing.T) {
	h := newTestHandler(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("expected text/html, got %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "<html>") {
		t.Fatalf("expected HTML body, got: %s", rec.Body.String())
	}
}

func TestSPAHandler_KnownAssetServedWithImmutableCache(t *testing.T) {
	h := newTestHandler(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/assets/main-abc123.js", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	cc := rec.Header().Get("Cache-Control")
	if !strings.Contains(cc, "immutable") {
		t.Fatalf("expected immutable cache header, got %q", cc)
	}
}

func TestSPAHandler_UnknownPathFallsBackToIndexHTML(t *testing.T) {
	h := newTestHandler(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/records/42", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for SPA fallback, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "<html>") {
		t.Fatalf("expected index.html body, got: %s", rec.Body.String())
	}
}

func TestSPAHandler_PostMethodNotHandled(t *testing.T) {
	h := newTestHandler(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-GET, got %d", rec.Code)
	}
}

func TestSPAHandler_LoginPathServesLoginHTML(t *testing.T) {
	h := newTestHandler(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("expected text/html, got %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "login") {
		t.Fatalf("expected login.html body, got: %s", rec.Body.String())
	}
}

func TestSPAHandler_LoginPathNotFallbackToIndexHTML(t *testing.T) {
	h := newTestHandler(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	h.ServeHTTP(rec, req)

	if strings.Contains(rec.Body.String(), "app") && !strings.Contains(rec.Body.String(), "login") {
		t.Fatal("/login should serve login.html, not index.html")
	}
}
