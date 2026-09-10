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

	"linkpulse/internal/analytics"
	"linkpulse/internal/apikey"
	"linkpulse/internal/auth"
	"linkpulse/internal/clickbuffer"
	"linkpulse/internal/config"
	"linkpulse/internal/link"
	"linkpulse/internal/publicapi"
	"linkpulse/internal/redirect"
	"linkpulse/internal/tenant"
)

// NewRouter builds the chi router with all global middleware and routes.
func NewRouter(
    logger *slog.Logger,
    db *pgxpool.Pool,
    cfg config.Config,
    clickBuf *clickbuffer.Buffer,
    clickSalt string,
) http.Handler {
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

    linkSvc := link.NewService(db, logger, cfg.AppBaseURL)
    linkHandler := link.NewHandler(linkSvc, logger)

    analyticsSvc := analytics.NewService(db, logger)
    analyticsHandler := analytics.NewHandler(analyticsSvc, logger)

    apikeySvc := apikey.NewService(db, logger)
    apikeyHandler := apikey.NewHandler(apikeySvc, logger)

    publicHandler := publicapi.NewHandler(db, linkSvc, logger)

    redirectSvc := redirect.NewService(db)
    redirectHandler := redirect.NewHandler(redirectSvc, clickBuf, logger, clickSalt, cfg.FrontendOrigin)

    // Public redirect engine: GET /{code} (PRD 9.5). Static routes
    // (healthz, api) always win over this parameter route.
    r.Get("/{code}", redirectHandler.Resolve)

    // Public API (PRD 9.8): authenticated by API keys, not sessions.
    // The required scope IS the authorization.
    r.Route("/api/v1/public", func(r chi.Router) {
        r.Use(apikeySvc.RequireKey)

        r.Group(func(r chi.Router) {
            r.Use(apikeySvc.RequireScope(apikey.ScopeLinksRead))
            r.Get("/links", publicHandler.ListLinks)
            r.Get("/links/{shortCode}", publicHandler.GetLink)
        })
        r.Group(func(r chi.Router) {
            r.Use(apikeySvc.RequireScope(apikey.ScopeLinksWrite))
            r.Post("/links", publicHandler.CreateLink)
            r.Patch("/links/{shortCode}", publicHandler.UpdateLink)
            r.Delete("/links/{shortCode}", publicHandler.DeleteLink)
        })
        r.Group(func(r chi.Router) {
            r.Use(apikeySvc.RequireScope(apikey.ScopeAnalyticsRead))
            r.Get("/links/{shortCode}/analytics", publicHandler.LinkAnalytics)
        })
    })

    r.Route("/api/v1", func(r chi.Router) {
        r.Route("/auth", func(r chi.Router) {
            // Public endpoints. Refresh & logout authenticate via the
            // refresh-token cookie, not a Bearer token.
            r.Post("/register", authHandler.Register)
            r.Post("/login", authHandler.Login)
            r.Post("/refresh", authHandler.Refresh)
            r.Post("/logout", authHandler.Logout)

            r.Group(func(r chi.Router) {
                r.Use(authSvc.RequireAuth)
                r.Get("/me", authHandler.Me)
            })
        })

        // Authenticated routes.
        r.Group(func(r chi.Router) {
            r.Use(authSvc.RequireAuth)

            // The invite code itself identifies the workspace, so
            // accepting is not tenant-scoped (PRD 9.3.2).
            r.Post("/invitations/accept", tenantHandler.AcceptInvite)

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

                    // Members (PRD 9.3.3–9.3.5).
                    r.Get("/members", tenantHandler.ListMembers)
                    r.Patch("/members/{userId}", tenantHandler.UpdateMemberRole)
                    r.Delete("/members/{userId}", tenantHandler.RemoveMember)
                    r.Post("/leave", tenantHandler.Leave)

                    // Invitations (PRD 9.3.1).
                    r.Post("/invitations", tenantHandler.CreateInvite)

                    // Analytics (PRD 9.6) — viewable by every member.
                    r.Get("/analytics/overview", analyticsHandler.Overview)

                    // API keys (PRD 9.7) — role checked in the handler.
                    r.Post("/api-keys", apikeyHandler.Create)
                    r.Get("/api-keys", apikeyHandler.List)
                    r.Delete("/api-keys/{keyId}", apikeyHandler.Revoke)

                    // Links (PRD 9.4). Reading is open to every member;
                    // write endpoints check the role inside the handler.
                    r.Route("/links", func(r chi.Router) {
                        r.Get("/", linkHandler.List)
                        r.Post("/", linkHandler.Create)
                        r.Get("/{linkId}", linkHandler.Get)
                        r.Patch("/{linkId}", linkHandler.Update)
                        r.Delete("/{linkId}", linkHandler.Delete)
                    })
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

// handleReadyz reports readiness: alive AND the database reachable.
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

// requestLogger logs one line per request (PRD 18.1).
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
func writeJSON(w http.ResponseWriter, status int, v any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(v)
}
