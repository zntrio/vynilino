package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"path/filepath"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	_ "modernc.org/sqlite" // SQLite driver

	migratesqlite "github.com/golang-migrate/migrate/v4/database/sqlite"

	"zntr.io/vynilino/internal/adapter/storage/sqlite/migrations"
)

// Open opens a SQLite database at the given path, enables WAL mode,
// runs pending migrations, and returns the connection.
// Background goroutines for disk metrics and WAL checkpoints run until ctx is cancelled.
func Open(ctx context.Context, path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// Tune connection pool for SQLite (single writer).
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	dbDir := filepath.Dir(path)
	go monitorDBDisk(ctx, dbDir)
	go runWALCheckpoint(ctx, db)

	return db, nil
}

// monitorDBDisk emits disk usage metrics for the SQLite directory every 60 s.
func monitorDBDisk(ctx context.Context, dir string) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			freeBytes, freePct, err := dbDiskFreeStats(dir)
			if err != nil {
				slog.Warn("disk stat failed", "dir", "db", "err", err)
				continue
			}
			slog.Info("disk_metric", "metric", "gauge:vynilino.storage.disk_free_bytes", "dir", "db", "value", freeBytes)
			slog.Info("disk_metric", "metric", "gauge:vynilino.storage.disk_free_pct", "dir", "db", "value", freePct)
			if freePct < 20 {
				slog.Warn("disk space low", "dir", "db", "free_pct", freePct)
			}
		}
	}
}

// runWALCheckpoint runs PRAGMA wal_checkpoint(TRUNCATE) every hour to bound WAL growth.
func runWALCheckpoint(ctx context.Context, db *sql.DB) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := db.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
				slog.Warn("wal checkpoint failed", "err", err)
			}
		}
	}
}

func dbDiskFreeStats(path string) (freeBytes uint64, freePct float64, err error) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, 0, err
	}
	free := uint64(st.Bavail) * uint64(st.Bsize)
	total := st.Blocks * uint64(st.Bsize)
	if total == 0 {
		return free, 0, nil
	}
	return free, float64(free) * 100.0 / float64(total), nil
}

func runMigrations(db *sql.DB) error {
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("iofs source: %w", err)
	}

	driver, err := migratesqlite.WithInstance(db, &migratesqlite.Config{})
	if err != nil {
		return fmt.Errorf("migrate driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "sqlite", driver)
	if err != nil {
		return fmt.Errorf("migrate instance: %w", err)
	}

	// Pre-check: abort if a previous run left the dirty flag set.
	// Manual fix: migrate -path <migrations-dir> -database <dsn> force <version>
	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("migration version check: %w", err)
	}
	if dirty {
		slog.Error("migration dirty state detected — manual intervention required",
			"version", version, "dirty", true)
		return fmt.Errorf("migration dirty state detected at version %d — "+
			"run: migrate force %d", version, version)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}

	version, dirty, _ = m.Version()
	slog.Info("migrations applied", "version", version, "dirty", dirty)
	return nil
}

// CheckMigrations reports whether there are pending migrations without applying them.
func CheckMigrations(path string) error {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	defer db.Close()

	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return err
	}

	driver, err := migratesqlite.WithInstance(db, &migratesqlite.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", src, "sqlite", driver)
	if err != nil {
		return err
	}

	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return err
	}
	slog.Info("migration status", "version", version, "dirty", dirty)
	return nil
}
