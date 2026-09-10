package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"linkpulse/internal/config"
	"linkpulse/internal/logutil"
	"linkpulse/internal/server"
	"linkpulse/internal/store/postgres"
)

func main() {
    // Load backend/.env if present. Real environment variables always
    // win (godotenv never overrides them), so production works without
    // a .env file.
    _ = godotenv.Load()

    cfg := config.Load()

    logger := logutil.New(logutil.Options{
        Level:  cfg.LogLevel,
        Format: cfg.LogFormat,
        Dir:    cfg.LogDir,
    })
    slog.SetDefault(logger)

    // Fail fast: no database URL or JWT secret, no point starting.
    if cfg.DatabaseURL == "" || cfg.JWTSecret == "" {
        slog.Error("DATABASE_URL and JWT_SECRET are required — set them in backend/.env")
        os.Exit(1)
    }

    // Connect to PostgreSQL and verify it answers a ping.
    pool, err := postgres.New(context.Background(), cfg.DatabaseURL)
    if err != nil {
        slog.Error("cannot connect to database", "error", err)
        os.Exit(1)
    }
    defer pool.Close()
    slog.Info("database connected")

    srv := &http.Server{
        Addr:         ":" + cfg.AppPort,
        Handler:      server.NewRouter(logger, pool, cfg),
        ReadTimeout:  10 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    go func() {
        slog.Info("server started", "addr", srv.Addr, "env", cfg.AppEnv)
        if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
            slog.Error("server error", "error", err)
            os.Exit(1)
        }
    }()

    // Block until Ctrl+C / terminate signal arrives.
    stop := make(chan os.Signal, 1)
    signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
    <-stop

    slog.Info("shutting down server gracefully...")
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        slog.Error("graceful shutdown failed", "error", err)
    }
    slog.Info("server stopped")
}
