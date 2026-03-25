package cmd

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	_ "modernc.org/sqlite" // SQLite driver
)

// BackupCmd returns the `backup` parent command group.
func BackupCmd() *cobra.Command {
	defaultDB := os.Getenv("VYNILINO_DB_PATH")
	if defaultDB == "" {
		defaultDB = "./vynilino.db"
	}

	var dbPath string

	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Manage database backups",
	}
	cmd.PersistentFlags().StringVar(&dbPath, "db", defaultDB, "path to SQLite database (default: $VYNILINO_DB_PATH or ./vynilino.db)")

	cmd.AddCommand(backupCreateCmd(&dbPath))
	cmd.AddCommand(backupVerifyCmd())

	return cmd
}

func backupCreateCmd(dbPath *string) *cobra.Command {
	var outputDir, hmacKey string

	c := &cobra.Command{
		Use:   "create",
		Short: "Create a verified backup of the database",
		RunE: func(cmd *cobra.Command, args []string) error {
			if hmacKey == "" {
				hmacKey = os.Getenv("VYNILINO_BACKUP_HMAC_KEY")
			}

			dir := outputDir
			if dir == "" {
				dir = filepath.Dir(*dbPath)
			}

			base := strings.TrimSuffix(filepath.Base(*dbPath), filepath.Ext(*dbPath))
			timestamp := time.Now().UTC().Format("20060102-150405")
			backupPath := filepath.Join(dir, base+"-"+timestamp+".db")

			db, err := sql.Open("sqlite", *dbPath)
			if err != nil {
				return fmt.Errorf("open source db: %w", err)
			}
			defer db.Close()

			if _, err := db.Exec("VACUUM main INTO ?", backupPath); err != nil {
				return fmt.Errorf("vacuum into %s: %w", backupPath, err)
			}

			var count int64
			if err := db.QueryRow("SELECT COUNT(*) FROM records").Scan(&count); err != nil {
				return fmt.Errorf("count records: %w", err)
			}

			sidecarPath := backupPath + ".count"
			if err := os.WriteFile(sidecarPath, []byte(strconv.FormatInt(count, 10)+"\n"), 0o600); err != nil {
				return fmt.Errorf("write sidecar %s: %w", sidecarPath, err)
			}

			// THREAT-014: optionally sign the backup file with HMAC-SHA256.
			if hmacKey != "" {
				if err := writeBackupHMAC(backupPath, hmacKey); err != nil {
					return fmt.Errorf("write backup signature: %w", err)
				}
			} else {
				slog.Warn("backup created without HMAC signature; set --hmac-key or VYNILINO_BACKUP_HMAC_KEY to enable authenticity verification")
			}

			slog.Info("backup_metric",
				"metric", "gauge:vynilino.backup.created",
				"result", "pass",
				"row_count", count,
				"path", backupPath,
				"signed", hmacKey != "",
			)
			return nil
		},
	}
	c.Flags().StringVar(&outputDir, "output", "", "destination directory for the backup (default: same directory as --db)")
	c.Flags().StringVar(&hmacKey, "hmac-key", "", "HMAC-SHA256 key for backup authenticity (default: $VYNILINO_BACKUP_HMAC_KEY)")
	return c
}

// writeBackupHMAC computes HMAC-SHA256 over the backup file and writes the hex digest to <path>.sig.
func writeBackupHMAC(backupPath, key string) error {
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("read backup: %w", err)
	}
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(data)
	sig := hex.EncodeToString(mac.Sum(nil))
	return os.WriteFile(backupPath+".sig", []byte(sig+"\n"), 0o600)
}

func backupVerifyCmd() *cobra.Command {
	var backupPath, hmacKey string

	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify backup integrity and row count",
		RunE: func(cmd *cobra.Command, args []string) error {
			if hmacKey == "" {
				hmacKey = os.Getenv("VYNILINO_BACKUP_HMAC_KEY")
			}

			result, actualCount, expectedCount, err := verifyBackup(backupPath, hmacKey)

			slog.Info("backup_metric",
				"metric", "gauge:vynilino.backup.verified",
				"result", result,
				"row_count", actualCount,
				"expected", expectedCount,
				"path", backupPath,
			)

			if err != nil {
				slog.Error("backup verification failed", "path", backupPath, "reason", err.Error())
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&backupPath, "backup", "", "path to the backup .db file to verify (required)")
	_ = cmd.MarkFlagRequired("backup")
	cmd.Flags().StringVar(&hmacKey, "hmac-key", "", "HMAC-SHA256 key for authenticity verification (default: $VYNILINO_BACKUP_HMAC_KEY)")

	return cmd
}

// verifyBackup runs integrity checks on the backup file at path.
// If key is non-empty, the HMAC-SHA256 signature in <path>.sig is also verified.
// It returns the result label ("pass" or "fail"), actual row count, expected row count, and any error.
func verifyBackup(path, key string) (result string, actualCount, expectedCount int64, err error) {
	db, openErr := sql.Open("sqlite", path+"?mode=ro")
	if openErr != nil {
		return "fail", 0, 0, fmt.Errorf("open backup: %w", openErr)
	}
	defer db.Close()

	// PRAGMA integrity_check returns one row per problem; a healthy DB returns a single "ok" row.
	rows, queryErr := db.Query("PRAGMA integrity_check")
	if queryErr != nil {
		return "fail", 0, 0, fmt.Errorf("integrity_check query: %w", queryErr)
	}
	defer rows.Close()

	for rows.Next() {
		var msg string
		if scanErr := rows.Scan(&msg); scanErr != nil {
			return "fail", 0, 0, fmt.Errorf("integrity_check scan: %w", scanErr)
		}
		if msg != "ok" {
			return "fail", 0, 0, fmt.Errorf("integrity_check: %s", msg)
		}
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return "fail", 0, 0, fmt.Errorf("integrity_check rows: %w", rowsErr)
	}

	if scanErr := db.QueryRow("SELECT COUNT(*) FROM records").Scan(&actualCount); scanErr != nil {
		return "fail", 0, 0, fmt.Errorf("count records: %w", scanErr)
	}

	sidecarPath := path + ".count"
	raw, readErr := os.ReadFile(sidecarPath)
	if readErr != nil {
		return "fail", actualCount, 0, fmt.Errorf("read sidecar %s: %w", sidecarPath, readErr)
	}

	expectedCount, parseErr := strconv.ParseInt(strings.TrimSpace(string(raw)), 10, 64)
	if parseErr != nil {
		return "fail", actualCount, 0, fmt.Errorf("parse sidecar count: %w", parseErr)
	}

	if actualCount != expectedCount {
		return "fail", actualCount, expectedCount,
			fmt.Errorf("row count mismatch: backup has %d rows, expected %d", actualCount, expectedCount)
	}

	// THREAT-014: verify HMAC-SHA256 signature when a key is provided.
	sigPath := path + ".sig"
	if key != "" {
		sigRaw, sigErr := os.ReadFile(sigPath)
		if sigErr != nil {
			return "fail", actualCount, expectedCount, fmt.Errorf("read signature file %s: %w", sigPath, sigErr)
		}
		expectedSig := strings.TrimSpace(string(sigRaw))

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return "fail", actualCount, expectedCount, fmt.Errorf("read backup for HMAC: %w", readErr)
		}
		mac := hmac.New(sha256.New, []byte(key))
		mac.Write(data)
		actualSig := hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(actualSig), []byte(expectedSig)) {
			return "fail", actualCount, expectedCount, fmt.Errorf("HMAC signature mismatch: backup may have been tampered with")
		}
	} else {
		if _, statErr := os.Stat(sigPath); statErr == nil {
			slog.Warn("signature file exists but no HMAC key provided; authenticity check skipped", "sig_path", sigPath)
		} else {
			slog.Warn("backup has no HMAC signature; set --hmac-key or VYNILINO_BACKUP_HMAC_KEY to enable authenticity verification")
		}
	}

	return "pass", actualCount, expectedCount, nil
}
