package filestore

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

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
func New(dir string) (*FileStore, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("create media dir: %w", err)
	}
	return &FileStore{root: dir}, nil
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

	filename := recordID + ext
	path := filepath.Join(dir, filename)
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
	ErrFileTooLarge        = fmt.Errorf("file exceeds maximum allowed size")
	ErrUnsupportedMediaType = fmt.Errorf("unsupported media type: only JPEG, PNG, WebP are accepted")
	ErrNotFound            = fmt.Errorf("not found")
)
