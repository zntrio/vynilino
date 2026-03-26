package graphql

import (
	"context"
	"net/http"
	"strings"

	"zntr.io/vynilino/internal/ctxutil"
)

// tokenValidator abstracts token validation so the middleware doesn't depend
// directly on the full UserService.
type tokenValidator interface {
	ValidateAccessToken(token string) (string, error)
}

// userLookup verifies that a user exists in the store and is active.
// It is a subset of domain.UserRepository used only by AuthMiddleware.
type userLookup interface {
	IsActiveUser(ctx context.Context, id string) bool
}

// AuthMiddleware extracts and validates the Bearer token from the Authorization
// header, injecting the user ID into the request context via ctxutil.
// After cryptographic validation it also verifies the user still exists in the
// store and has not been deactivated.
// Unauthenticated requests pass through — resolvers enforce auth themselves.
func AuthMiddleware(validator tokenValidator, users userLookup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token != "" {
				if userID, err := validator.ValidateAccessToken(token); err == nil {
					if users.IsActiveUser(r.Context(), userID) {
						r = r.WithContext(ctxutil.WithUserID(r.Context(), userID))
					}
				}
			}
			if bt := r.Header.Get("X-Bootstrap-Token"); bt != "" {
				r = r.WithContext(ctxutil.WithBootstrapToken(r.Context(), bt))
			}
			next.ServeHTTP(w, r)
		})
	}
}

// UserIDFromContext is a convenience re-export for handlers in this package.
func UserIDFromContext(r *http.Request) (string, bool) {
	return ctxutil.UserIDFromContext(r.Context())
}

func extractBearerToken(r *http.Request) string {
	// Authorization header takes precedence (API clients, curl, etc.).
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	// Fall back to the HttpOnly cookie set by the browser (THREAT-006 mitigation).
	if c, err := r.Cookie("vynilino_access"); err == nil {
		return c.Value
	}
	return ""
}
