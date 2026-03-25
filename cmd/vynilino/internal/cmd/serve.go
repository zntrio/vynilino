package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/fx"

	"zntr.io/vynilino/internal/adapter/discogs"
	"zntr.io/vynilino/internal/adapter/filestore"
	httpgql "zntr.io/vynilino/internal/adapter/graphql"
	"zntr.io/vynilino/internal/adapter/storage/sqlite"
	"zntr.io/vynilino/internal/app"
	"zntr.io/vynilino/internal/config"
	"zntr.io/vynilino/internal/domain"
	"zntr.io/vynilino/web"
)

// ServeCmd returns the `serve` subcommand that starts the HTTP server.
func ServeCmd() *cobra.Command {
	var checkMigrations bool

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			setupLogger(cfg)

			if checkMigrations {
				if err := sqlite.CheckMigrations(cfg.DBPath); err != nil {
					return fmt.Errorf("migration check: %w", err)
				}
				slog.Info("migrations up to date")
				return nil
			}

			fx.New(
				fx.Supply(cfg),
				fx.Provide(
					newDB,
					newFileStore,
					newStaticFS,
					sqlite.NewUserRepository,
					sqlite.NewRecordRepository,
					sqlite.NewTokenRepository,
					sqlite.NewOIDCIdentityRepository,
					sqlite.NewOIDCStateRepository,
					app.NewEventBus,
					app.NewRecordService,
					newUserService,
					app.NewOIDCService,
					newDiscogsClient,
					app.NewDiscogsService,
					httpgql.NewRouter,
					newHTTPServer,
				),
				fx.Invoke(startServer),
			).Run()
			return nil
		},
	}
	cmd.Flags().BoolVar(&checkMigrations, "check-migrations", false, "Check pending migrations and exit")
	return cmd
}

func newDB(cfg *config.Config) (*sql.DB, error) {
	return sqlite.Open(cfg.DBPath)
}

func newFileStore(cfg *config.Config) (*filestore.FileStore, error) {
	return filestore.New(cfg.MediaDir)
}

func newStaticFS() fs.FS {
	return web.StaticFS()
}

func newDiscogsClient(cfg *config.Config) (app.DiscogsSearcher, error) {
	return discogs.New(cfg.DiscogsToken, "vynilino/1.0 +https://github.com/zntr-io/vynilino")
}

func newUserService(
	users domain.UserRepository,
	tokens domain.TokenRepository,
	cfg *config.Config,
) (*app.UserService, error) {
	return app.NewUserService(users, tokens, cfg.TokenKey, cfg.SingleOwner)
}

func newHTTPServer(handler http.Handler, cfg *config.Config) *http.Server {
	return &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}

func startServer(lc fx.Lifecycle, srv *http.Server) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			slog.Info("server listening", "addr", srv.Addr)
			go func() {
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					slog.Error("server error", "err", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			slog.Info("shutting down server")
			shutCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			return srv.Shutdown(shutCtx)
		},
	})
}

func setupLogger(cfg *config.Config) {
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	var handler slog.Handler
	if cfg.IsDevelopment() {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(handler))
}
