package httpx

import (
    "context"

    "github.com/google/uuid"
)

// userIDKey is the context key for the authenticated caller's ID.
type userIDKey struct{}

// WithUserID stores the authenticated user ID in the request context.
// Set by the auth middleware; read by every handler that needs the
// caller. Living here (not in the auth package) keeps the dependency
// graph acyclic: any module can read the caller without importing auth.
func WithUserID(ctx context.Context, userID uuid.UUID) context.Context {
    return context.WithValue(ctx, userIDKey{}, userID)
}

// UserIDFrom extracts the authenticated user ID set by the auth middleware.
func UserIDFrom(ctx context.Context) (uuid.UUID, bool) {
    id, ok := ctx.Value(userIDKey{}).(uuid.UUID)
    return id, ok
}
