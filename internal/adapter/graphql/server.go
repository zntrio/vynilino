package graphql

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/google/uuid"
	"github.com/vektah/gqlparser/v2/ast"

	"zntr.io/vynilino/internal/adapter/filestore"
	"zntr.io/vynilino/internal/adapter/graphql/graph"
	uiadapter "zntr.io/vynilino/internal/adapter/ui"
	"zntr.io/vynilino/internal/app"
	"zntr.io/vynilino/internal/config"
	"zntr.io/vynilino/internal/ctxutil"
	"zntr.io/vynilino/internal/domain"
)

const maxRequestBodyBytes = 1 << 20 // 1 MB

// NewRouter builds the chi router with all middleware and routes.
func NewRouter(
	cfg *config.Config,
	db *sql.DB,
	userSvc *app.UserService,
	recordSvc *app.RecordService,
	bus *app.EventBus,
	discogsSvc *app.DiscogsService,
	userRepo domain.UserRepository,
	fileStore *filestore.FileStore,
	oidcSvc *app.OIDCService,
	staticFiles fs.FS,
) http.Handler {
	r := chi.NewRouter()

	// ── Base middleware ──────────────────────────────────────────────────────
	// Only trust forwarding headers when the server is explicitly placed behind
	// a known reverse proxy (THREAT-005 mitigation).
	if cfg.BehindProxy {
		r.Use(middleware.RealIP)
	}
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(securityHeadersMiddleware(cfg.IsDevelopment()))
	// Store the ResponseWriter in context so resolvers can set cookies.
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(ctxutil.WithResponseWriter(r.Context(), w)))
		})
	})
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Bootstrap-Token"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(AuthMiddleware(userSvc))

	// ── Health ───────────────────────────────────────────────────────────────
	r.Get("/health", newHealthHandler(db, fileStore))

	// ── GraphQL ──────────────────────────────────────────────────────────────
	gqlHandler := newGraphQLHandler(cfg, userSvc, recordSvc, discogsSvc, userRepo, oidcSvc)
	r.With(httprate.LimitByIP(20, time.Minute)).Handle("/graphql", http.MaxBytesHandler(gqlHandler, maxRequestBodyBytes))

	if cfg.Playground {
		// The GraphiQL playground HTML embeds inline scripts and loads CDN resources,
		// so it requires a relaxed CSP. The override is applied only to this single
		// route; the global strict nonce-based CSP is unaffected.
		r.With(playgroundCSPOverride).Get("/playground", playground.Handler("vynilino", "/graphql"))
	}

	// ── Media ─────────────────────────────────────────────────────────────────
	// THREAT-007 / THREAT-OTHER-001: all media routes require authentication.
	r.Route("/media", func(r chi.Router) {
		r.Use(requireAuthMiddleware)
		r.Post("/cover-art", uploadCoverArtHandler(recordSvc, fileStore))
		r.Get("/cover-art/{userID}/{filename}", serveCoverArtHandler(fileStore))
	})

	// ── Export / Import ───────────────────────────────────────────────────────
	r.Route("/export", func(r chi.Router) {
		r.Use(requireAuthMiddleware)
		r.Get("/json", exportJSONHandler(recordSvc))
		r.Get("/csv", exportCSVHandler(recordSvc))
	})
	r.Route("/import", func(r chi.Router) {
		r.Use(requireAuthMiddleware)
		r.Post("/csv", importCSVHandler(recordSvc, bus))
	})

	// ── UI (must be last — SPA fallback catches all remaining GET paths) ──────
	uiHandler := uiadapter.New(staticFiles, userRepo, fileStore)
	r.Route("/api", func(r chi.Router) {
		r.Get("/me", uiHandler.MeHandler())
		r.With(requireAuthMiddleware).Post("/upload", uiHandler.UploadHandler())
	})
	r.Get("/login", loginRedirectHandler(cfg, oidcSvc, uiHandler.LoginHandler()))
	r.Mount("/", uiHandler.SPAHandler())

	return r
}

func newGraphQLHandler(
	cfg *config.Config,
	userSvc *app.UserService,
	recordSvc *app.RecordService,
	discogsSvc *app.DiscogsService,
	userRepo domain.UserRepository,
	oidcSvc *app.OIDCService,
) http.Handler {
	resolver := &graph.Resolver{
		UserSvc:    userSvc,
		RecordSvc:  recordSvc,
		DiscogsSvc: discogsSvc,
		UserRepo:   userRepo,
		OIDCSvc:    oidcSvc,
	}
	schema := graph.NewExecutableSchema(graph.Config{Resolvers: resolver})
	srv := handler.New(schema)

	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
		// Authenticate the WebSocket connection at the connection_init stage
		// (THREAT-010 mitigation). If the HTTP upgrade request already carried
		// a valid cookie, AuthMiddleware will have set the userID in ctx; we
		// accept that directly. Otherwise we validate a Bearer token from the
		// connection_init payload so that non-browser API clients still work.
		InitFunc: func(ctx context.Context, initPayload transport.InitPayload) (context.Context, *transport.InitPayload, error) {
			if _, ok := ctxutil.UserIDFromContext(ctx); ok {
				return ctx, &initPayload, nil
			}
			authHeader, _ := initPayload["Authorization"].(string)
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token == "" {
				return ctx, nil, fmt.Errorf("unauthorized")
			}
			userID, err := userSvc.ValidateAccessToken(token)
			if err != nil {
				return ctx, nil, fmt.Errorf("unauthorized")
			}
			return ctxutil.WithUserID(ctx, userID), &initPayload, nil
		},
	})

	srv.SetQueryCache(queryDocumentCache{})

	if cfg.Introspection {
		srv.Use(extension.Introspection{})
	}
	srv.Use(extension.FixedComplexityLimit(100))

	return srv
}

// queryDocumentCache is a simple no-op query cache.
type queryDocumentCache struct{}

func (queryDocumentCache) Add(_ context.Context, _ string, _ *ast.QueryDocument) {}
func (queryDocumentCache) Get(_ context.Context, _ string) (*ast.QueryDocument, bool) {
	return nil, false
}

// playgroundCSPOverride replaces the Content-Security-Policy header with a
// permissive policy that allows the GraphiQL playground's CDN scripts and
// inline event handlers. It must only be applied to the /playground route.
func playgroundCSPOverride(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Override the nonce-based CSP set by securityHeadersMiddleware.
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-inline' https://unpkg.com https://cdn.jsdelivr.net; "+
				"style-src 'self' 'unsafe-inline' https://unpkg.com https://cdn.jsdelivr.net; "+
				"img-src 'self' data:; "+
				"connect-src 'self' ws: wss:; "+
				"font-src 'self' https://unpkg.com https://cdn.jsdelivr.net")
		next.ServeHTTP(w, r)
	})
}

// requireAuthMiddleware returns 401 if no user is in the context.
func requireAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := ctxutil.UserIDFromContext(r.Context()); !ok {
			http.Error(w, `{"error":"UNAUTHENTICATED"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func securityHeadersMiddleware(isDev bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Generate a per-request 128-bit nonce encoded as base64url (no padding).
			b := make([]byte, 16)
			if _, err := rand.Read(b); err != nil {
				http.Error(w, "internal error", http.StatusInternalServerError)
				return
			}
			nonce := base64.RawURLEncoding.EncodeToString(b)

			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "no-referrer")
			w.Header().Set("Content-Security-Policy",
				"default-src 'self'; "+
					"script-src 'self' 'nonce-"+nonce+"'; "+
					"style-src 'self' 'nonce-"+nonce+"'; "+
					"img-src 'self' data: blob:; "+
					"connect-src 'self' ws: wss:; "+
					"font-src 'self'")
			// Emit HSTS in production so browsers enforce HTTPS (THREAT-009 mitigation).
			if !isDev {
				w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			}

			next.ServeHTTP(w, r.WithContext(ctxutil.WithCSPNonce(r.Context(), nonce)))
		})
	}
}

// DBPinger can check database connectivity.
type DBPinger interface {
	PingContext(ctx context.Context) error
}

// NewHealthHandler returns a health handler that checks DB connectivity.
func NewHealthHandler(db DBPinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbStatus := "ok"
		if err := db.PingContext(r.Context()); err != nil {
			dbStatus = "error"
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","db":"` + dbStatus + `"}`))
	}
}

// newHealthHandler returns a health handler that checks DB connectivity and disk usage.
func newHealthHandler(db *sql.DB, fs *filestore.FileStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbStatus := "ok"
		if err := db.PingContext(r.Context()); err != nil {
			dbStatus = "error"
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		diskPct, _ := fs.DiskFreePercent()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","db":%q,"disk_free_pct":%.1f}`, dbStatus, diskPct)
	}
}

// uploadCoverArtHandler handles POST /media/cover-art.
func uploadCoverArtHandler(recordSvc *app.RecordService, fs *filestore.FileStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, _ := ctxutil.UserIDFromContext(r.Context())

		if err := r.ParseMultipartForm(filestore.MaxCoverArtBytes + 1); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "missing file field", http.StatusBadRequest)
			return
		}
		defer file.Close()

		recordID := r.FormValue("recordId")
		if recordID == "" {
			http.Error(w, "missing recordId", http.StatusBadRequest)
			return
		}
		if _, err := uuid.Parse(recordID); err != nil {
			http.Error(w, "invalid recordId", http.StatusBadRequest)
			return
		}

		// THREAT-OTHER-001: verify the record exists and belongs to the caller.
		if _, err := recordSvc.GetByID(r.Context(), recordID, userID); err != nil {
			http.Error(w, "record not found", http.StatusNotFound)
			return
		}

		url, err := fs.StoreCoverArt(userID, recordID, file)
		if err != nil {
			switch err {
			case filestore.ErrFileTooLarge:
				http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
			case filestore.ErrUnsupportedMediaType:
				http.Error(w, err.Error(), http.StatusUnsupportedMediaType)
			case filestore.ErrInvalidPath:
				http.Error(w, "invalid recordId", http.StatusBadRequest)
			case filestore.ErrInsufficientSpace:
				http.Error(w, err.Error(), http.StatusInsufficientStorage)
			default:
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"url":"` + url + `"}`))
	}
}

// serveCoverArtHandler handles GET /media/cover-art/{userID}/{filename}.
// THREAT-007: auth is enforced by the enclosing route group; the caller must
// own the resource (userID path param must match the authenticated user).
func serveCoverArtHandler(fs *filestore.FileStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ownerID := chi.URLParam(r, "userID")
		filename := chi.URLParam(r, "filename")

		callerID, _ := ctxutil.UserIDFromContext(r.Context())
		if callerID != ownerID {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		f, mime, err := fs.OpenCoverArt(ownerID, filename)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		defer f.Close()

		w.Header().Set("Content-Type", mime)
		w.Header().Set("Cache-Control", "private, max-age=86400")
		http.ServeContent(w, r, filename, time.Time{}, f)
	}
}
