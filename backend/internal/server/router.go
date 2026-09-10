// Package server wires the HTTP router, middleware, and route handlers.
package server

import (
    "context"
    "encoding/json"
    "fmt"
    "log/slog"
    "net/http"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
    "github.com/go-chi/cors"
    "github.com/jackc/pgx/v5/pgxpool"

    "linkpulse/internal/auth"
    "linkpulse/internal/config"
    "linkpulse/internal/tenant"
)

// NewRouter builds the chi router with all global middleware and routes.
func NewRouter(logger *slog.Logger, db *pgxpool.Pool, cfg config.Config) http.Handler {
    r := chi.NewRouter()

    // Global middleware (runs on every request, in this order).
    r.Use(middleware.RequestID)
    r.Use(middleware.RealIP)
    r.Use(requestLogger(logger))
    r.Use(middleware.Recoverer)

    // CORS: allow ONLY our frontend origin. The origin list is explicit —
    // a wildcard combined with AllowCredentials would let ANY site send
    // credentialed requests (PRD 16.17; also a red line in the protocol).
    r.Use(cors.Handler(cors.Options{
        AllowedOrigins:   []string{cfg.FrontendOrigin},
        AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
        AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization"},
        ExposedHeaders:   []string{"X-Request-Id"},
        AllowCredentials: true,
        MaxAge:           300,
    }))

    // Health probes (PRD 18.3).
    r.Get("/healthz", handleHealthz)
    r.Get("/readyz", handleReadyz(db))

    // Feature handlers.
    authSvc := auth.NewService(db, auth.ServiceConfig{
        JWTSecret:    cfg.JWTSecret,
        AccessTTL:    cfg.JWTAccessTTL,
        RefreshTTL:   cfg.JWTRefreshTTL,
        CookieSecure: cfg.AppEnv != "development",
    })
    authHandler := auth.NewHandler(authSvc, logger)

    tenantSvc := tenant.NewService(db, logger)
    tenantHandler := tenant.NewHandler(tenantSvc, logger)

    r.Route("/api/v1", func(r chi.Router) {
        r.Route("/auth", func(r chi.Router) {
            // Public endpoints. Refresh & logout authenticate via the
            // refresh-token cookie, not a Bearer token.
            r.Post("/register", authHandler.Register)
            r.Post("/login", authHandler.Login)
            r.Post("/refresh", authHandler.Refresh)
            r.Post("/logout", authHandler.Logout)

            // Token-protected endpoints.
            r.Group(func(r chi.Router) {
                r.Use(authSvc.RequireAuth)
                r.Get("/me", authHandler.Me)
            })
        })

        // Authenticated, workspace-scoped routes.
        r.Group(func(r chi.Router) {
            r.Use(authSvc.RequireAuth)

            r.Route("/tenants", func(r chi.Router) {
                // Collection endpoints (no tenant in the URL yet).
                r.Post("/", tenantHandler.Create)
                r.Get("/", tenantHandler.List)

                r.Route("/{tenantId}", func(r chi.Router) {
                    // Membership gate: everything below requires the
                    // caller to be a member of this workspace.
                    r.Use(tenantSvc.RequireMembership())

                    r.Get("/", tenantHandler.Get)
                    r.Patch("/", tenantHandler.Update)
                })
            })
        })
    })

    return r
}

// handleHealthz reports liveness. Always 200 while the process runs.
func handleHealthz(w http.ResponseWriter, r *http.Request) {
    writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleReadyz reports readiness: the process is alive AND the database
// is reachable. Returns 503 when the database is down.
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

// requestLogger logs one line per request (PRD 18.1). The human-readable
// summary (method, path, status, duration) is the message so the console
// stays one short line; the request ID rides along as a structured
// attribute for the JSON file log.
func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()

            ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

            // PRD 18.2: return the request ID in the response header.
            if reqID := middleware.GetReqID(r.Context()); reqID != "" {
                ww.Header().Set("X-Request-Id", reqID)
            }

            next.ServeHTTP(ww, r)

            logger.Info(
                fmt.Sprintf("%s %s → %d (%dms)",
                    r.Method, r.URL.Path, ww.Status(), time.Since(start).Milliseconds()),
                "request_id", middleware.GetReqID(r.Context()),
            )
        })
    }
}

// writeJSON writes v as a JSON response with the given status code.
// Health probes use this raw format; API endpoints use httpx instead.
func writeJSON(w http.ResponseWriter, status int, v any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(v)
}
