package graphql

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"

	"zntr.io/vynilino/internal/adapter/filestore"
	"zntr.io/vynilino/internal/ctxutil"
)

func TestSecurityHeadersMiddleware_CSPNonce(t *testing.T) {
	mw := securityHeadersMiddleware(false)

	var capturedNonce string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedNonce = ctxutil.CSPNonceFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := mw(inner)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	csp := rr.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("expected Content-Security-Policy header")
	}
	if strings.Contains(csp, "unsafe-inline") {
		t.Errorf("CSP must not contain 'unsafe-inline', got: %s", csp)
	}
	if strings.Contains(csp, "unsafe-eval") {
		t.Errorf("CSP must not contain 'unsafe-eval', got: %s", csp)
	}
	if capturedNonce == "" {
		t.Fatal("expected nonce to be set in context")
	}
	if !strings.Contains(csp, "'nonce-"+capturedNonce+"'") {
		t.Errorf("CSP must contain nonce %q, got: %s", capturedNonce, csp)
	}
}

func TestSecurityHeadersMiddleware_NoncesAreUnique(t *testing.T) {
	mw := securityHeadersMiddleware(false)

	nonces := make(map[string]struct{})
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nonces[ctxutil.CSPNonceFromContext(r.Context())] = struct{}{}
	})
	handler := mw(inner)

	for range 10 {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
	if len(nonces) != 10 {
		t.Errorf("expected 10 unique nonces, got %d", len(nonces))
	}
}

func TestProcessCSVImport_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel

	csvData := "title,artist\nDark Side,Pink Floyd\nWall,Pink Floyd\n"
	_, _, _, cancelled := processCSVImport(ctx, nil, nil, "user1", strings.NewReader(csvData))
	if !cancelled {
		t.Fatal("expected cancelled=true for a pre-cancelled context")
	}
}

func TestImportCSVHandler_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "import.csv")
	_, _ = fw.Write([]byte("title,artist\nDark Side,Pink Floyd\n"))
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/import/csv", &body).WithContext(
		ctxutil.WithUserID(ctx, "user1"),
	)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()

	importCSVHandler(nil, nil).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["cancelled"] != true {
		t.Fatalf("expected cancelled=true, got %v", resp["cancelled"])
	}
}

func TestImportCSVHandler_ConcurrentImport(t *testing.T) {
	// Simulate a concurrent import already in progress for this user.
	importInProgress.Store("user1", struct{}{})
	defer importInProgress.Delete("user1")

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "import.csv")
	_, _ = fw.Write([]byte("title,artist\nTest,Artist\n"))
	mw.Close()

	ctx := ctxutil.WithUserID(context.Background(), "user1")
	req := httptest.NewRequest(http.MethodPost, "/import/csv", &body).WithContext(ctx)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()

	importCSVHandler(nil, nil).ServeHTTP(rr, req)

	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 when import already in progress, got %d", rr.Code)
	}
}

func newRateLimitedHandler() http.Handler {
	r := chi.NewRouter()
	r.With(httprate.LimitByIP(20, time.Minute)).Post("/graphql", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	return r
}

func TestGraphQL_IPRateLimit(t *testing.T) {
	h := newRateLimitedHandler()

	for i := range 25 {
		req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if i < 20 && rr.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i+1, rr.Code)
		}
		if i >= 20 && rr.Code != http.StatusTooManyRequests {
			t.Fatalf("request %d: expected 429, got %d", i+1, rr.Code)
		}
	}
}

func TestGraphQL_IPRateLimit_DifferentIP(t *testing.T) {
	h := newRateLimitedHandler()

	// Saturate IP 1.2.3.4 past the limit.
	for range 21 {
		req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
	}

	// A different IP must still succeed.
	req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
	req.RemoteAddr = "5.6.7.8:1234"
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for second IP, got %d", rr.Code)
	}
}

func TestUploadCoverArtHandler_InvalidRecordID(t *testing.T) {
	fs, err := filestore.New(t.Context(), t.TempDir())
	if err != nil {
		t.Fatalf("new file store: %v", err)
	}
	handler := uploadCoverArtHandler(nil, fs)

	cases := []string{
		"../../etc/passwd",
		"not-a-uuid",
		"../sibling",
	}
	for _, recordID := range cases {
		t.Run(recordID, func(t *testing.T) {
			var body bytes.Buffer
			mw := multipart.NewWriter(&body)
			_ = mw.WriteField("recordId", recordID)
			fw, _ := mw.CreateFormFile("file", "cover.jpg")
			// Minimal JPEG bytes so the file field is present.
			fw.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00,
				0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00})
			mw.Close()

			req := httptest.NewRequest(http.MethodPost, "/media/cover-art", &body)
			req.Header.Set("Content-Type", mw.FormDataContentType())
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Fatalf("recordId=%q: expected 400, got %d", recordID, rr.Code)
			}
		})
	}
}
