// Package ctxutil provides shared context key helpers used across adapters.
package ctxutil

import (
	"context"
	"net/http"
)

type contextKey string

const userIDKey contextKey = "userID"

// WithUserID returns a derived context with the user ID embedded.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserIDFromContext retrieves the authenticated user's ID from the context.
func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(userIDKey).(string)
	return id, ok && id != ""
}

const bootstrapTokenKey contextKey = "bootstrapToken"

// WithBootstrapToken returns a derived context with the bootstrap token embedded.
func WithBootstrapToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, bootstrapTokenKey, token)
}

// BootstrapTokenFromContext retrieves the bootstrap token from the context.
func BootstrapTokenFromContext(ctx context.Context) string {
	v, _ := ctx.Value(bootstrapTokenKey).(string)
	return v
}

const cspNonceKey contextKey = "cspNonce"

// WithCSPNonce returns a derived context carrying the per-request CSP nonce.
func WithCSPNonce(ctx context.Context, nonce string) context.Context {
	return context.WithValue(ctx, cspNonceKey, nonce)
}

// CSPNonceFromContext retrieves the CSP nonce from the context.
// Returns an empty string if no nonce is present.
func CSPNonceFromContext(ctx context.Context) string {
	v, _ := ctx.Value(cspNonceKey).(string)
	return v
}

// ClientInfo holds per-request client metadata for audit logging.
type ClientInfo struct {
	IP        string
	UserAgent string
}

const clientInfoKey contextKey = "clientInfo"

// WithClientInfo returns a derived context carrying the caller's IP and User-Agent.
func WithClientInfo(ctx context.Context, ip, userAgent string) context.Context {
	return context.WithValue(ctx, clientInfoKey, ClientInfo{IP: ip, UserAgent: userAgent})
}

// ClientInfoFromContext retrieves client metadata from the context.
// Returns a zero-value ClientInfo (empty strings) when not present.
func ClientInfoFromContext(ctx context.Context) ClientInfo {
	v, _ := ctx.Value(clientInfoKey).(ClientInfo)
	return v
}

const responseWriterKey contextKey = "responseWriter"

// WithResponseWriter returns a derived context carrying the HTTP response writer.
// This allows resolvers and services deeper in the call stack to set cookies.
func WithResponseWriter(ctx context.Context, w http.ResponseWriter) context.Context {
	return context.WithValue(ctx, responseWriterKey, w)
}

// ResponseWriterFromContext retrieves the HTTP response writer from the context.
func ResponseWriterFromContext(ctx context.Context) (http.ResponseWriter, bool) {
	w, ok := ctx.Value(responseWriterKey).(http.ResponseWriter)
	return w, ok
}
