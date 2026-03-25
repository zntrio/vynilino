package cmd_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"zntr.io/vynilino/cmd/vynilino/internal/cmd"
	"zntr.io/vynilino/internal/adapter/storage/sqlite"
)

// newTestSQLiteDB creates a temporary SQLite database with migrations applied
// and returns its path.
func newTestSQLiteDB(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "vynilino-backup-test-*.db")
	if err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	f.Close()

	db, err := sqlite.Open(context.Background(), f.Name())
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	return f.Name()
}

func runBackupCreate(t *testing.T, dbPath, outputDir string) {
	t.Helper()
	root := cmd.BackupCmd()
	root.SetArgs([]string{"create", "--db", dbPath, "--output", outputDir})
	if err := root.Execute(); err != nil {
		t.Fatalf("backup create: %v", err)
	}
}

func findBackupFile(t *testing.T, dir string) string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read backup dir: %v", err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".db") {
			return filepath.Join(dir, e.Name())
		}
	}
	t.Fatal("no backup .db file found in", dir)
	return ""
}

// TestBackupCreateAndVerify creates a backup and verifies it successfully.
func TestBackupCreateAndVerify(t *testing.T) {
	dbPath := newTestSQLiteDB(t)
	outputDir := t.TempDir()

	runBackupCreate(t, dbPath, outputDir)

	backupPath := findBackupFile(t, outputDir)

	root := cmd.BackupCmd()
	root.SetArgs([]string{"verify", "--backup", backupPath})
	if err := root.Execute(); err != nil {
		t.Fatalf("backup verify: %v", err)
	}
}

// TestBackupVerifyCorrupted corrupts the backup bytes and expects verify to fail.
func TestBackupVerifyCorrupted(t *testing.T) {
	dbPath := newTestSQLiteDB(t)
	outputDir := t.TempDir()

	runBackupCreate(t, dbPath, outputDir)

	backupPath := findBackupFile(t, outputDir)

	// Overwrite the header bytes to corrupt the file.
	f, err := os.OpenFile(backupPath, os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("open backup for corruption: %v", err)
	}
	if _, err := f.WriteAt([]byte("CORRUPTED!!!!!!!"), 0); err != nil {
		f.Close()
		t.Fatalf("corrupt backup: %v", err)
	}
	f.Close()

	root := cmd.BackupCmd()
	root.SetArgs([]string{"verify", "--backup", backupPath})
	if err := root.Execute(); err == nil {
		t.Fatal("expected verify to fail on corrupted backup, got nil error")
	}
}

// TestBackupVerifyCountMismatch tampers the .count sidecar and expects verify to fail.
func TestBackupVerifyCountMismatch(t *testing.T) {
	dbPath := newTestSQLiteDB(t)
	outputDir := t.TempDir()

	runBackupCreate(t, dbPath, outputDir)

	backupPath := findBackupFile(t, outputDir)
	sidecarPath := backupPath + ".count"

	if err := os.WriteFile(sidecarPath, []byte("99999\n"), 0o600); err != nil {
		t.Fatalf("tamper sidecar: %v", err)
	}

	root := cmd.BackupCmd()
	root.SetArgs([]string{"verify", "--backup", backupPath})
	if err := root.Execute(); err == nil {
		t.Fatal("expected verify to fail on count mismatch, got nil error")
	}
}

// TestBackupVerifyMissingSidecar deletes the .count sidecar and expects verify to fail.
func TestBackupVerifyMissingSidecar(t *testing.T) {
	dbPath := newTestSQLiteDB(t)
	outputDir := t.TempDir()

	runBackupCreate(t, dbPath, outputDir)

	backupPath := findBackupFile(t, outputDir)
	sidecarPath := backupPath + ".count"

	if err := os.Remove(sidecarPath); err != nil {
		t.Fatalf("remove sidecar: %v", err)
	}

	root := cmd.BackupCmd()
	root.SetArgs([]string{"verify", "--backup", backupPath})
	if err := root.Execute(); err == nil {
		t.Fatal("expected verify to fail on missing sidecar, got nil error")
	}
}
