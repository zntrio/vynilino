package graphql

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"zntr.io/vynilino/internal/config"
)

// stubLoginPage is a simple handler that serves a minimal login page response.
func stubLoginPage(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte("<html>login</html>"))
}

// stubOIDC is a test double for oidcURLProvider.
type stubOIDC struct {
	url string
	err error
}

func (s *stubOIDC) AuthorizationURL(_ context.Context) (string, string, error) {
	return s.url, "state", s.err
}

func TestLoginRedirectHandler_FlagOff(t *testing.T) {
	cfg := &config.Config{OIDCAutoRedirect: false}
	h := loginRedirectHandler(cfg, nil, stubLoginPage)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	h(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "login") {
		t.Fatal("expected login.html content in response body")
	}
}

func TestLoginRedirectHandler_FlagOnNilSvc(t *testing.T) {
	cfg := &config.Config{OIDCAutoRedirect: true}
	h := loginRedirectHandler(cfg, nil, stubLoginPage)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	h(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("want 200 (OIDC svc nil), got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "login") {
		t.Fatal("expected login.html content in response body")
	}
}

func TestLoginRedirectHandler_FlagOnSvcSuccess(t *testing.T) {
	cfg := &config.Config{OIDCAutoRedirect: true}
	svc := &stubOIDC{url: "https://idp.example.com/auth?foo=bar"}
	h := loginRedirectHandler(cfg, svc, stubLoginPage)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	h(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("want 302, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != svc.url {
		t.Fatalf("want Location %q, got %q", svc.url, loc)
	}
	if rec.Body.Len() > 0 && !strings.Contains(rec.Body.String(), "302") {
		// http.Redirect writes a small HTML body — that's fine, login page not sent
		if strings.Contains(rec.Body.String(), "login") {
			t.Fatal("login page HTML must not be in the 302 response body")
		}
	}
}

func TestLoginRedirectHandler_FlagOnSvcError(t *testing.T) {
	cfg := &config.Config{OIDCAutoRedirect: true}
	svc := &stubOIDC{err: errors.New("provider unreachable")}
	h := loginRedirectHandler(cfg, svc, stubLoginPage)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	h(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "OIDC provider unavailable") {
		t.Fatalf("expected error message in body, got: %s", rec.Body.String())
	}
}
