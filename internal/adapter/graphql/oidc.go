package graphql

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"zntr.io/vynilino/internal/app"
	"zntr.io/vynilino/internal/domain"
)

// oidcCookieName is the HttpOnly cookie that carries the PASETO access token.
// Must match accessCookieName in internal/adapter/graphql/graph/schema.resolvers.go
// and the literal in internal/adapter/graphql/middleware.go.
const oidcCookieName = "vynilino_access"

// oidcCallbackSvc is the subset of app.OIDCService used by oidcCallbackHandler.
type oidcCallbackSvc interface {
	HandleCallback(ctx context.Context, code, stateVal string) (*app.TokenPair, error)
}

// oidcAuthorizeHandler handles GET /oidc/authorize.
// It generates a fresh PKCE authorization URL and redirects the browser to
// the OIDC provider to begin the login flow.
func oidcAuthorizeHandler(svc oidcURLProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if svc == nil {
			http.Redirect(w, r, "/login?error=oidc_not_configured", http.StatusFound)
			return
		}
		authURL, _, err := svc.AuthorizationURL(r.Context())
		if err != nil {
			http.Redirect(w, r, "/login?error=oidc_unavailable", http.StatusFound)
			return
		}
		http.Redirect(w, r, authURL, http.StatusFound)
	}
}

// oidcCallbackHandler handles GET /oidc/callback.
// It receives the authorization code and state from the OIDC provider,
// exchanges them for a Vynilino token pair, sets the HttpOnly access
// cookie, then redirects the browser to the application root.
func oidcCallbackHandler(svc oidcCallbackSvc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if svc == nil {
			http.Redirect(w, r, "/login?error=oidc_not_configured", http.StatusFound)
			return
		}

		// OIDC provider may return an error (e.g. user denied consent).
		if errParam := r.URL.Query().Get("error"); errParam != "" {
			http.Redirect(w, r, "/login?error="+url.QueryEscape(errParam), http.StatusFound)
			return
		}

		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")

		pair, err := svc.HandleCallback(r.Context(), code, state)
		if err != nil {
			redirect := "/login?error=auth_failed"
			if errors.Is(err, domain.ErrOIDCUserForbidden) {
				redirect = "/login?error=access_denied"
			}
			http.Redirect(w, r, redirect, http.StatusFound)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     oidcCookieName,
			Value:    pair.AccessToken,
			Path:     "/",
			MaxAge:   pair.ExpiresIn,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
		})
		http.Redirect(w, r, "/", http.StatusFound)
	}
}
