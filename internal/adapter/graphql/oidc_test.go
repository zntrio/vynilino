package graphql

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"zntr.io/vynilino/internal/app"
)

// stubCallbackSvc is a test double for oidcCallbackSvc.
type stubCallbackSvc struct {
	pair *app.TokenPair
	err  error
}

func (s *stubCallbackSvc) HandleCallback(_ context.Context, _, _ string) (*app.TokenPair, error) {
	return s.pair, s.err
}

// ── oidcAuthorizeHandler ──────────────────────────────────────────────────────

func TestOIDCAuthorizeHandler_NilSvc(t *testing.T) {
	h := oidcAuthorizeHandler(nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oidc/authorize", nil)
	h(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("want 302, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/login?error=oidc_not_configured" {
		t.Fatalf("unexpected Location: %q", loc)
	}
}

func TestOIDCAuthorizeHandler_SvcError(t *testing.T) {
	svc := &stubOIDC{err: errors.New("provider down")}
	h := oidcAuthorizeHandler(svc)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oidc/authorize", nil)
	h(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("want 302, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/login?error=oidc_unavailable" {
		t.Fatalf("unexpected Location: %q", loc)
	}
}

func TestOIDCAuthorizeHandler_Success(t *testing.T) {
	svc := &stubOIDC{url: "https://idp.example.com/auth?client_id=x"}
	h := oidcAuthorizeHandler(svc)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oidc/authorize", nil)
	h(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("want 302, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != svc.url {
		t.Fatalf("want Location %q, got %q", svc.url, loc)
	}
}

// ── oidcCallbackHandler ──────────────────────────────────────────────────────

func TestOIDCCallbackHandler_NilSvc(t *testing.T) {
	h := oidcCallbackHandler(nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oidc/callback?code=abc&state=xyz", nil)
	h(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("want 302, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/login?error=oidc_not_configured" {
		t.Fatalf("unexpected Location: %q", loc)
	}
}

func TestOIDCCallbackHandler_ProviderError(t *testing.T) {
	svc := &stubCallbackSvc{}
	h := oidcCallbackHandler(svc)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oidc/callback?error=access_denied", nil)
	h(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("want 302, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/login?error=access_denied" {
		t.Fatalf("unexpected Location: %q", loc)
	}
}

func TestOIDCCallbackHandler_ProviderErrorEncoded(t *testing.T) {
	svc := &stubCallbackSvc{}
	h := oidcCallbackHandler(svc)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oidc/callback?error=temporarily+unavailable", nil)
	h(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("want 302, got %d", rec.Code)
	}
	loc := rec.Header().Get("Location")
	if loc == "" {
		t.Fatal("expected Location header")
	}
	// Must not contain unencoded spaces in the error value.
	if loc == "/login?error=temporarily unavailable" {
		t.Fatalf("error value was not URL-encoded: %q", loc)
	}
}

func TestOIDCCallbackHandler_CallbackError(t *testing.T) {
	svc := &stubCallbackSvc{err: errors.New("invalid state")}
	h := oidcCallbackHandler(svc)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oidc/callback?code=abc&state=xyz", nil)
	h(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("want 302, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/login?error=auth_failed" {
		t.Fatalf("unexpected Location: %q", loc)
	}
}

func TestOIDCCallbackHandler_Success(t *testing.T) {
	svc := &stubCallbackSvc{
		pair: &app.TokenPair{
			AccessToken: "v4.local.test-token",
			ExpiresIn:   900,
		},
	}
	h := oidcCallbackHandler(svc)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/oidc/callback?code=abc&state=xyz", nil)
	h(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("want 302, got %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/" {
		t.Fatalf("want Location /, got %q", loc)
	}

	var found *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == oidcCookieName {
			found = c
		}
	}
	if found == nil {
		t.Fatalf("expected %q cookie to be set", oidcCookieName)
	}
	if found.Value != svc.pair.AccessToken {
		t.Fatalf("cookie value: want %q, got %q", svc.pair.AccessToken, found.Value)
	}
	if !found.HttpOnly {
		t.Error("cookie must be HttpOnly")
	}
	if !found.Secure {
		t.Error("cookie must be Secure")
	}
	if found.SameSite != http.SameSiteStrictMode {
		t.Errorf("cookie SameSite: want Strict, got %v", found.SameSite)
	}
	if found.Path != "/" {
		t.Errorf("cookie Path: want /, got %q", found.Path)
	}
	if found.MaxAge != svc.pair.ExpiresIn {
		t.Errorf("cookie MaxAge: want %d, got %d", svc.pair.ExpiresIn, found.MaxAge)
	}
}
