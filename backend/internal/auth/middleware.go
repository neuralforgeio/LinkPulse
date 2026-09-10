package auth

import (
	"net/http"
	"strings"

	"linkpulse/internal/httpx"
)

// RequireAuth rejects requests without a valid Bearer access token and,
// on success, stores the user ID in the request context via
// httpx.WithUserID — shared storage, so every module can read the caller
// without importing this package.
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

        next.ServeHTTP(w, r.WithContext(httpx.WithUserID(r.Context(), userID)))
    })
}
