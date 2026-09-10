package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewRouter(logger *slog.Logger, db *pgxpool.Pool) http.Handler {
	r := chi.NewRouter();

	r.Use(middleware.RequestID) // assigns X-Request-Id header + context
	r.Use(middleware.RealIP)		// resolves real client IP behind proxies
	r.Use(requestLogger(logger)) // structured slog per request
	r.Use(middleware.Recoverer)

	// Livenses probe: is the process alive?
	r.Get("/healthz", handleHealthz)

	// Readiness probe: alive AND the database answers a ping
	r.Get("/readyz", handleReadyz(db))

	return r
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleReadyz(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "db_unreachable"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}
}

// requestLogger logs on structured line per request: request_id
// method, path, status, duration
func requestLogger(logger *slog.Logger) func (http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// wrap the writer so we can read the final status code
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			logger.Info("http request", "request_id", middleware.GetReqID(r.Context()), "method", r.Method, "path", r.URL.Path, "status", ww.Status(), "duration_ms", time.Since(start).Milliseconds())
		})
	}
}

// writeJSON writes v as a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Status is already sent; nothing more we can do on encode failure.
	_ = json.NewEncoder(w).Encode(v)
}
