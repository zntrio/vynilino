package filestore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// statfs is a package-level variable to allow test injection.
var statfs = func(path string, stat *syscall.Statfs_t) error {
	return syscall.Statfs(path, stat)
}

// allowedMIME maps accepted MIME types to their canonical file extension.
var allowedMIME = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// MaxCoverArtBytes is the maximum allowed cover art file size (5 MB).
const MaxCoverArtBytes = 5 << 20

// FileStore handles local filesystem storage for cover art.
type FileStore struct {
	root string
}

// New creates a FileStore rooted at dir, creating it if needed.
// It starts a background goroutine that emits disk usage metrics every 60 s;
// the goroutine exits when ctx is cancelled.
func New(ctx context.Context, dir string) (*FileStore, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("create media dir: %w", err)
	}
	s := &FileStore{root: dir}
	go s.monitorDisk(ctx)
	return s, nil
}

// monitorDisk emits disk usage metrics every 60 s until ctx is cancelled.
func (s *FileStore) monitorDisk(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			freeBytes, freePct, err := diskFreeStats(s.root)
			if err != nil {
				slog.Warn("disk stat failed", "dir", "media", "err", err)
				continue
			}
			slog.Info("disk_metric", "metric", "gauge:vynilino.storage.disk_free_bytes", "dir", "media", "value", freeBytes)
			slog.Info("disk_metric", "metric", "gauge:vynilino.storage.disk_free_pct", "dir", "media", "value", freePct)
			if freePct < 20 {
				slog.Warn("disk space low", "dir", "media", "free_pct", freePct)
			}
		}
	}
}

// DiskFreePercent returns the percentage of free disk space for the media directory.
func (s *FileStore) DiskFreePercent() (float64, error) {
	_, pct, err := diskFreeStats(s.root)
	return pct, err
}

// diskFreeStats returns free bytes and free percentage for the given path.
func diskFreeStats(path string) (freeBytes uint64, freePct float64, err error) {
	var st syscall.Statfs_t
	if err = statfs(path, &st); err != nil {
		return 0, 0, err
	}
	free := uint64(st.Bavail) * uint64(st.Bsize)
	total := st.Blocks * uint64(st.Bsize)
	if total == 0 {
		return free, 0, nil
	}
	return free, float64(free) * 100.0 / float64(total), nil
}

// StoreCoverArt saves a cover art upload, returning the relative URL path.
// The caller must limit the reader to maxCoverArtBytes before passing it in.
func (s *FileStore) StoreCoverArt(userID, recordID string, r io.Reader) (string, error) {
	// Read up to MaxCoverArtBytes+1 to detect oversized uploads.
	buf := make([]byte, MaxCoverArtBytes+1)
	n, err := io.ReadFull(r, buf)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("read cover art: %w", err)
	}
	data := buf[:n]
	if n > MaxCoverArtBytes {
		return "", ErrFileTooLarge
	}

	mime := http.DetectContentType(data)
	// Trim parameters (e.g. "image/jpeg; charset=...")
	if idx := strings.Index(mime, ";"); idx >= 0 {
		mime = strings.TrimSpace(mime[:idx])
	}
	ext, ok := allowedMIME[mime]
	if !ok {
		return "", ErrUnsupportedMediaType
	}

	dir := filepath.Join(s.root, userID)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("create user media dir: %w", err)
	}

	// Reject recordIDs containing path separators (catches URL-encoded traversal too).
	if strings.ContainsAny(recordID, `/\`) {
		slog.Warn("path traversal rejected", "outcome", "path_traversal_rejected",
			"user_id", userID, "record_id_raw", recordID)
		return "", ErrInvalidPath
	}

	filename := recordID + ext
	path := filepath.Join(dir, filename)
	rel, err := filepath.Rel(dir, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		slog.Warn("path traversal rejected", "outcome", "path_traversal_rejected",
			"user_id", userID, "record_id_raw", recordID)
		return "", ErrInvalidPath
	}

	// Pre-write disk space guard: reject if available < file size + 10 MB reserve.
	const reserveBytes = 10 << 20
	var st syscall.Statfs_t
	if err := statfs(dir, &st); err == nil {
		available := uint64(st.Bavail) * uint64(st.Bsize)
		if available < uint64(len(data))+reserveBytes {
			return "", ErrInsufficientSpace
		}
	}

	if err := os.WriteFile(path, data, 0o640); err != nil {
		return "", fmt.Errorf("write cover art: %w", err)
	}

	return "/media/cover-art/" + userID + "/" + filename, nil
}

// DeleteCoverArt removes all cover art files for the given record.
func (s *FileStore) DeleteCoverArt(userID, recordID string) error {
	dir := filepath.Join(s.root, userID)
	for ext := range allowedMIME {
		_ = ext
	}
	// Remove any file matching recordID.* pattern.
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	prefix := recordID + "."
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), prefix) {
			_ = os.Remove(filepath.Join(dir, e.Name()))
		}
	}
	return nil
}

// OpenCoverArt opens a cover art file for serving.
// Returns the file, MIME type, and any error.
func (s *FileStore) OpenCoverArt(userID, filename string) (*os.File, string, error) {
	// Sanitise: filename must not contain path separators.
	if strings.ContainsAny(filename, `/\`) {
		return nil, "", ErrNotFound
	}
	path := filepath.Join(s.root, userID, filename)
	// Verify the resolved path is still under root.
	rel, err := filepath.Rel(s.root, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return nil, "", ErrNotFound
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, "", ErrNotFound
		}
		return nil, "", err
	}

	// Detect MIME from first 512 bytes.
	header := make([]byte, 512)
	nr, _ := f.Read(header)
	mime := http.DetectContentType(header[:nr])
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		f.Close()
		return nil, "", err
	}

	return f, mime, nil
}

var (
	ErrFileTooLarge         = fmt.Errorf("file exceeds maximum allowed size")
	ErrUnsupportedMediaType = fmt.Errorf("unsupported media type: only JPEG, PNG, WebP are accepted")
	ErrNotFound             = fmt.Errorf("not found")
	ErrInvalidPath          = fmt.Errorf("path traversal attempt rejected")
	ErrInsufficientSpace    = fmt.Errorf("insufficient disk space: less than 10 MB available")
)
