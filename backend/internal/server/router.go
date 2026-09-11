// Package server wires the HTTP router, middleware, and route handlers.
package server

import (
    "context"
    "encoding/json"
    "fmt"
    "log/slog"
    "net/http"
    "strings"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
    "github.com/go-chi/cors"
    "github.com/jackc/pgx/v5/pgxpool"

    "linkpulse/internal/analytics"
    "linkpulse/internal/apikey"
    "linkpulse/internal/audit"
    "linkpulse/internal/auth"
    "linkpulse/internal/clickbuffer"
    "linkpulse/internal/config"
    "linkpulse/internal/httpx"
    "linkpulse/internal/link"
    "linkpulse/internal/publicapi"
    "linkpulse/internal/ratelimit"
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

    // Security headers (PRD 16.18).
    r.Use(func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.Header().Set("X-Content-Type-Options", "nosniff")
            w.Header().Set("X-Frame-Options", "DENY")
            w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
            next.ServeHTTP(w, r)
        })
    })

    // Global middleware.
    r.Use(middleware.RequestID)
    r.Use(middleware.RealIP)
    r.Use(requestLogger(logger))
    r.Use(middleware.Recoverer)

    // CORS allowlist: FRONTEND_ORIGIN supports a comma-separated list.
    allowedOrigins := strings.Split(cfg.FrontendOrigin, ",")
    for i := range allowedOrigins {
        allowedOrigins[i] = strings.TrimSpace(allowedOrigins[i])
    }
    publicFrontend := allowedOrigins[len(allowedOrigins)-1]

    r.Use(cors.Handler(cors.Options{
        AllowedOrigins: allowedOrigins,
        AllowedMethods: []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
        AllowedHeaders: []string{"Accept", "Content-Type", "Authorization", "ngrok-skip-browser-warning"},
        ExposedHeaders: []string{"X-Request-Id", "X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset", "Retry-After"},
        AllowCredentials: true,
        MaxAge:           300,
    }))

    // Health probes (PRD 18.3).
    r.Get("/healthz", handleHealthz)
    r.Get("/readyz", handleReadyz(db))

    // Rate limiters (PRD 9.9).
    rlLimit := func(n int) int {
        if cfg.RateLimitEnabled {
            return n
        }
        return 0
    }
    redirectRL := ratelimit.New(rlLimit(cfg.RateLimitRedirectPerMinute), time.Minute)
    loginRL := ratelimit.New(rlLimit(cfg.RateLimitLoginPerMinute), time.Minute)
    registerRL := ratelimit.New(rlLimit(cfg.RateLimitRegisterPerMinute), time.Minute)
    publicRL := ratelimit.New(rlLimit(cfg.RateLimitPublicAPIPerMinute), time.Minute)
    dashboardRL := ratelimit.New(rlLimit(cfg.RateLimitDashboardPerMinute), time.Minute)

    // Feature handlers.
    authSvc := auth.NewService(db, auth.ServiceConfig{
        JWTSecret:         cfg.JWTSecret,
        AccessTTL:         cfg.JWTAccessTTL,
        RefreshTTL:        cfg.JWTRefreshTTL,
        CookieSecure:      cfg.AppEnv != "development",
        CookieSameSiteNone: strings.EqualFold(cfg.CookieSameSite, "none"),
    })
    authHandler := auth.NewHandler(authSvc, logger)

    // Reset links point at the local frontend in development, the
    // public deployment otherwise.
    resetLinkBase := publicFrontend
    if cfg.AppEnv == "development" && len(allowedOrigins) > 0 {
        resetLinkBase = allowedOrigins[0]
    }
    resetHandler := auth.NewResetHandler(db, logger, resetLinkBase)

    tenantSvc := tenant.NewService(db, logger)
    tenantHandler := tenant.NewHandler(tenantSvc, logger)

    linkSvc := link.NewService(db, logger, cfg.AppBaseURL)
    linkHandler := link.NewHandler(linkSvc, logger)

    analyticsSvc := analytics.NewService(db, logger)
    analyticsHandler := analytics.NewHandler(analyticsSvc, logger)

    apikeySvc := apikey.NewService(db, logger)
    apikeyHandler := apikey.NewHandler(apikeySvc, logger)

    auditSvc := audit.NewService(db, logger, clickSalt)
    auditHandler := audit.NewHandler(auditSvc, logger)

    publicHandler := publicapi.NewHandler(db, linkSvc, logger)

    redirectSvc := redirect.NewService(db)
    redirectHandler := redirect.NewHandler(redirectSvc, clickBuf, logger, redirect.HandlerConfig{
        IPSalt:             clickSalt,
        FrontendOrigin:     publicFrontend,
        JWTSecret:          cfg.JWTSecret,
        CookieSecure:       cfg.AppEnv != "development",
        CookieSameSiteNone: strings.EqualFold(cfg.CookieSameSite, "none"),
    })

    // Public redirect engine: GET /{code}, IP rate limited (PRD 9.5, 9.9).
    r.With(redirectRL.Middleware(ratelimit.KeyIP)).
        Get("/{code}", redirectHandler.Resolve)

    // Public API (PRD 9.8).
    r.Route("/api/v1/public", func(r chi.Router) {
        // Visitor endpoint: link password verification — NO API key
        // (PRD 9.5.3).
        r.With(redirectRL.Middleware(ratelimit.KeyIP)).
            Post("/links/{shortCode}/verify-password", redirectHandler.VerifyPassword)

        r.Group(func(r chi.Router) {
            r.Use(apikeySvc.RequireKey)
            r.Use(publicRL.Middleware(func(r *http.Request) string {
                if a, ok := apikey.FromContext(r.Context()); ok {
                    return a.KeyID.String()
                }
                return ratelimit.KeyIP(r)
            }))
            r.Use(auditSvc.TrackMutations)

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
    })

    r.Route("/api/v1", func(r chi.Router) {
        r.Route("/auth", func(r chi.Router) {
            r.With(registerRL.Middleware(ratelimit.KeyIP)).
                With(auditSvc.TrackAction("auth.register", "user")).
                Post("/register", authHandler.Register)
            r.With(loginRL.Middleware(ratelimit.KeyIP)).
                With(auditSvc.TrackAction("auth.login", "user")).
                Post("/login", authHandler.Login)
            r.With(auditSvc.TrackAction("auth.logout", "user")).
                Post("/logout", authHandler.Logout)
            r.Post("/refresh", authHandler.Refresh)

            // Password reset — tight IP limit: anti email-enumeration
            // (PRD 9.1.7, 9.1.8, 9.9).
            r.With(registerRL.Middleware(ratelimit.KeyIP)).
                Post("/password/reset-request", resetHandler.RequestReset)
            r.With(registerRL.Middleware(ratelimit.KeyIP)).
                Post("/password/reset-confirm", resetHandler.ConfirmReset)

            r.Group(func(r chi.Router) {
                r.Use(authSvc.RequireAuth)
                r.Get("/me", authHandler.Me)
                r.Patch("/me", authHandler.UpdateProfile)
            })
        })

        r.Group(func(r chi.Router) {
            r.Use(authSvc.RequireAuth)
            r.Use(dashboardRL.Middleware(func(r *http.Request) string {
                if id, ok := httpx.UserIDFrom(r.Context()); ok {
                    return id.String()
                }
                return ratelimit.KeyIP(r)
            }))

            r.With(auditSvc.TrackAction("invitation.accept", "invitation")).
                Post("/invitations/accept", tenantHandler.AcceptInvite)

            r.Route("/tenants", func(r chi.Router) {
                r.With(auditSvc.TrackAction("tenant.create", "tenant")).
                    Post("/", tenantHandler.Create)
                r.Get("/", tenantHandler.List)

                r.Route("/{tenantId}", func(r chi.Router) {
                    r.Use(tenantSvc.RequireMembership())
                    r.Use(auditSvc.TrackMutations)

                    r.Get("/", tenantHandler.Get)
                    r.Patch("/", tenantHandler.Update)

                    r.Get("/members", tenantHandler.ListMembers)
                    r.Patch("/members/{userId}", tenantHandler.UpdateMemberRole)
                    r.Delete("/members/{userId}", tenantHandler.RemoveMember)
                    r.Post("/leave", tenantHandler.Leave)

                    r.Post("/invitations", tenantHandler.CreateInvite)

                    r.Get("/analytics/overview", analyticsHandler.Overview)

                    r.Post("/api-keys", apikeyHandler.Create)
                    r.Get("/api-keys", apikeyHandler.List)
                    r.Delete("/api-keys/{keyId}", apikeyHandler.Revoke)

                    r.Get("/audit-logs", auditHandler.List)

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
