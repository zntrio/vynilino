package graphql_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	httpgql "zntr.io/vynilino/internal/adapter/graphql"
	"zntr.io/vynilino/internal/ctxutil"
)

type stubValidator struct {
	userID string
	err    error
}

func (s *stubValidator) ValidateAccessToken(_ string) (string, error) {
	return s.userID, s.err
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	stub := &stubValidator{userID: "user-123"}
	mw := httpgql.AuthMiddleware(stub)

	var gotID string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID, _ = ctxutil.UserIDFromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if gotID != "user-123" {
		t.Fatalf("expected userID=user-123, got %q", gotID)
	}
}

func TestAuthMiddleware_MissingToken(t *testing.T) {
	stub := &stubValidator{userID: "user-123"}
	mw := httpgql.AuthMiddleware(stub)

	var gotID string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID, _ = ctxutil.UserIDFromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil) // no Authorization header
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if gotID != "" {
		t.Fatalf("expected empty userID for missing token, got %q", gotID)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	stub := &stubValidator{err: &invalidTokenError{}}
	mw := httpgql.AuthMiddleware(stub)

	var gotID string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID, _ = ctxutil.UserIDFromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer bad-token")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if gotID != "" {
		t.Fatalf("expected empty userID for invalid token, got %q", gotID)
	}
}

type invalidTokenError struct{}

func (e *invalidTokenError) Error() string { return "invalid token" }
