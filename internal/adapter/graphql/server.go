package graphql

import (
	"context"
	"io/fs"
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
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
	userSvc *app.UserService,
	recordSvc *app.RecordService,
	discogsSvc *app.DiscogsService,
	userRepo domain.UserRepository,
	fileStore *filestore.FileStore,
	oidcSvc *app.OIDCService,
	staticFiles fs.FS,
) http.Handler {
	r := chi.NewRouter()

	// ── Base middleware ──────────────────────────────────────────────────────
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(securityHeadersMiddleware)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(AuthMiddleware(userSvc))

	// ── Health ───────────────────────────────────────────────────────────────
	r.Get("/health", healthHandler()) // basic; wire DB-aware handler in production

	// ── GraphQL ──────────────────────────────────────────────────────────────
	gqlHandler := newGraphQLHandler(cfg, userSvc, recordSvc, discogsSvc, userRepo, oidcSvc)
	r.Handle("/graphql", http.MaxBytesHandler(gqlHandler, maxRequestBodyBytes))

	if cfg.Playground {
		r.Get("/playground", playground.Handler("vynilino", "/graphql"))
	}

	// ── Media ─────────────────────────────────────────────────────────────────
	r.Route("/media", func(r chi.Router) {
		r.With(requireAuthMiddleware).Post("/cover-art", uploadCoverArtHandler(fileStore))
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
		r.Post("/csv", importCSVHandler(recordSvc))
	})

	// ── UI (must be last — SPA fallback catches all remaining GET paths) ──────
	uiHandler := uiadapter.New(staticFiles, userRepo, fileStore)
	r.Route("/api", func(r chi.Router) {
		r.Get("/me", uiHandler.MeHandler())
		r.With(requireAuthMiddleware).Post("/upload", uiHandler.UploadHandler())
	})
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
	})

	srv.SetQueryCache(queryDocumentCache{})

	if cfg.Introspection {
		srv.Use(extension.Introspection{})
	}
	srv.Use(extension.FixedComplexityLimit(1000))

	return srv
}

// queryDocumentCache is a simple no-op query cache.
type queryDocumentCache struct{}

func (queryDocumentCache) Add(_ context.Context, _ string, _ *ast.QueryDocument) {}
func (queryDocumentCache) Get(_ context.Context, _ string) (*ast.QueryDocument, bool) {
	return nil, false
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

func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; connect-src 'self' ws: wss:; font-src 'self'")
		next.ServeHTTP(w, r)
	})
}

// DBPinger can check database connectivity.
type DBPinger interface {
	PingContext(ctx context.Context) error
}

// NewHealthHandler returns a health handler that also checks DB connectivity.
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

func healthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","db":"ok"}`))
	}
}

// uploadCoverArtHandler handles POST /media/cover-art.
func uploadCoverArtHandler(fs *filestore.FileStore) http.HandlerFunc {
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

		url, err := fs.StoreCoverArt(userID, recordID, file)
		if err != nil {
			switch err {
			case filestore.ErrFileTooLarge:
				http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
			case filestore.ErrUnsupportedMediaType:
				http.Error(w, err.Error(), http.StatusUnsupportedMediaType)
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
// No auth is required: the URL already encodes both a user UUID and a record
// UUID, making it an unguessable capability URL safe to load from <img> tags.
func serveCoverArtHandler(fs *filestore.FileStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ownerID := chi.URLParam(r, "userID")
		filename := chi.URLParam(r, "filename")

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

