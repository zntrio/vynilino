// Package ctxutil provides shared context key helpers used across adapters.
package ctxutil

import "context"

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
