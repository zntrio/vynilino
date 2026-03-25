package graphql

import (
	"context"
	"net/http"

	"zntr.io/vynilino/internal/config"
)

const oidcUnavailableHTML = `<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><title>Login unavailable</title></head>
<body>
<p>OIDC provider unavailable. Contact your administrator.</p>
</body>
</html>`

// oidcURLProvider is the subset of app.OIDCService used by loginRedirectHandler.
// *app.OIDCService satisfies this interface.
type oidcURLProvider interface {
	AuthorizationURL(ctx context.Context) (url, state string, err error)
}

// loginRedirectHandler returns an http.HandlerFunc for GET /login.
//
// When OIDCAutoRedirect is enabled and svc is non-nil, it generates a fresh
// OIDC authorization URL and issues a 302 redirect. On failure it returns 500
// with a plain error page — there is no fallback to the login form.
//
// Otherwise it delegates to loginPage, which is expected to serve login.html
// with a per-request CSP nonce injected.
func loginRedirectHandler(cfg *config.Config, svc oidcURLProvider, loginPage http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cfg.OIDCAutoRedirect && svc != nil {
			authURL, _, err := svc.AuthorizationURL(r.Context())
			if err != nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(oidcUnavailableHTML))
				return
			}
			http.Redirect(w, r, authURL, http.StatusFound)
			return
		}

		loginPage(w, r)
	}
}
