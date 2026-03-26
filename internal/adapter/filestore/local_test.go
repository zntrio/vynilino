package filestore_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"zntr.io/vynilino/internal/adapter/filestore"
)

func newTestStore(t *testing.T) *filestore.FileStore {
	t.Helper()
	fs, err := filestore.New(t.Context(), t.TempDir(), time.Minute)
	if err != nil {
		t.Fatalf("new file store: %v", err)
	}
	return fs
}

// minimalJPEG is the smallest valid JPEG (SOI + EOI markers).
var minimalJPEG = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00,
	0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00}

func TestStoreCoverArt_JPEG(t *testing.T) {
	fs := newTestStore(t)
	url, err := fs.StoreCoverArt("user-1", "record-1", bytes.NewReader(minimalJPEG))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(url, "record-1") {
		t.Fatalf("URL should contain record ID: %q", url)
	}
}

func TestStoreCoverArt_UnsupportedType(t *testing.T) {
	fs := newTestStore(t)
	txt := []byte("this is plain text not an image")
	_, err := fs.StoreCoverArt("user-1", "record-2", bytes.NewReader(txt))
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
	if err != filestore.ErrUnsupportedMediaType {
		t.Fatalf("expected ErrUnsupportedMediaType, got %v", err)
	}
}

func TestStoreCoverArt_TooLarge(t *testing.T) {
	fs := newTestStore(t)
	// Create a fake JPEG header followed by padding to exceed 5MB.
	data := make([]byte, filestore.MaxCoverArtBytes+1)
	copy(data, minimalJPEG)
	_, err := fs.StoreCoverArt("user-1", "record-3", bytes.NewReader(data))
	if err == nil {
		t.Fatal("expected error for oversized file")
	}
	if err != filestore.ErrFileTooLarge {
		t.Fatalf("expected ErrFileTooLarge, got %v", err)
	}
}

func TestDeleteCoverArt(t *testing.T) {
	fs := newTestStore(t)
	_, err := fs.StoreCoverArt("user-1", "record-4", bytes.NewReader(minimalJPEG))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	if err := fs.DeleteCoverArt("user-1", "record-4"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	// Serving should now return not found.
	_, _, err = fs.OpenCoverArt("user-1", "record-4.jpg")
	if err == nil {
		t.Fatal("expected not found after deletion")
	}
}

func TestOpenCoverArt_PathTraversal(t *testing.T) {
	fs := newTestStore(t)
	_, _, err := fs.OpenCoverArt("user-1", "../../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal attempt")
	}
}

func TestStoreCoverArt_InsufficientSpace(t *testing.T) {
	fs := newTestStore(t)

	// Simulate a full disk: 0 available blocks, block size 4096, 1000 total blocks.
	orig := *filestore.StatfsFunc
	*filestore.StatfsFunc = filestore.FakeStatfs(0, 4096, 1000)
	t.Cleanup(func() { *filestore.StatfsFunc = orig })

	_, err := fs.StoreCoverArt("user-1", "record-5", bytes.NewReader(minimalJPEG))
	if err == nil {
		t.Fatal("expected error for insufficient disk space")
	}
	if !errors.Is(err, filestore.ErrInsufficientSpace) {
		t.Fatalf("expected ErrInsufficientSpace, got %v", err)
	}
}

func TestStoreCoverArt_PathTraversal(t *testing.T) {
	cases := []string{
		"../../etc/passwd",
		"../sibling",
		"%2e%2e/escape",
	}
	for _, recordID := range cases {
		t.Run(recordID, func(t *testing.T) {
			fs := newTestStore(t)
			_, err := fs.StoreCoverArt("user-1", recordID, bytes.NewReader(minimalJPEG))
			if err == nil {
				t.Fatal("expected error for path traversal attempt")
			}
			if err != filestore.ErrInvalidPath {
				t.Fatalf("expected ErrInvalidPath, got %v", err)
			}
		})
	}
}
