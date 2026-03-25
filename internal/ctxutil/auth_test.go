package ctxutil

import (
	"context"
	"testing"
)

func TestCSPNonce_RoundTrip(t *testing.T) {
	ctx := WithCSPNonce(context.Background(), "abc123")
	if got := CSPNonceFromContext(ctx); got != "abc123" {
		t.Fatalf("expected %q, got %q", "abc123", got)
	}
}

func TestCSPNonce_Missing(t *testing.T) {
	if got := CSPNonceFromContext(context.Background()); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestCSPNonce_IsolatedFromParent(t *testing.T) {
	parent := context.Background()
	child := WithCSPNonce(parent, "nonce-value")

	if CSPNonceFromContext(parent) != "" {
		t.Fatal("nonce leaked into parent context")
	}
	if CSPNonceFromContext(child) != "nonce-value" {
		t.Fatal("nonce not found in child context")
	}
}
