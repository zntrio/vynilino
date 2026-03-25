package ui_test

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"zntr.io/vynilino/internal/adapter/ui"
	"zntr.io/vynilino/internal/ctxutil"
)

// stubFS builds a minimal in-memory FS with index.html, login.html, and a hashed asset.
func stubFS() fs.FS {
	return fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte(
			`<html><head><script type="module" src="/assets/main.js"></script></head><body>app</body></html>`,
		)},
		"login.html": &fstest.MapFile{Data: []byte(
			`<html><head><script type="module" src="/assets/login.js"></script></head><body>login</body></html>`,
		)},
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

// /login is now handled by an explicit chi route (loginRedirectHandler) registered
// before the SPA mount. SPAHandler never sees GET /login in production; when called
// directly it falls through to the index.html SPA fallback.
func TestSPAHandler_LoginPathFallsBackToIndexHTML(t *testing.T) {
	h := newTestHandler(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "app") {
		t.Fatalf("expected index.html (SPA fallback), got: %s", rec.Body.String())
	}
}

func TestSPAHandler_NonceInjectedIntoIndexHTML(t *testing.T) {
	h := ui.New(stubFS(), nil, nil).SPAHandler()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(ctxutil.WithCSPNonce(req.Context(), "testnonce123"))
	h.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, `nonce="testnonce123"`) {
		t.Errorf("expected nonce attribute in HTML, got: %s", body)
	}
}

func TestSPAHandler_NoStoreCacheOnIndexHTML(t *testing.T) {
	h := newTestHandler(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	h.ServeHTTP(rec, req)

	cc := rec.Header().Get("Cache-Control")
	if cc != "no-store" {
		t.Fatalf("expected Cache-Control: no-store on index.html, got %q", cc)
	}
}
