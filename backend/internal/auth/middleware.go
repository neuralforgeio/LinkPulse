package auth

import (
	"context"
	"linkpulse/internal/httpx"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type userIDKey struct{}

func (s *Service) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const prefix = "Bearer "

		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, prefix) {
			httpx.Error(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "missing bearer token")
			return
		}

		userID, err := ParseAccessToken(s.jwtSecret, strings.TrimPrefix(header, prefix))
		if err != nil {
			httpx.Error(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "invalid or expired token")
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey{}, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserIDFrom(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userIDKey{}).(uuid.UUID)
	return id, ok
}
