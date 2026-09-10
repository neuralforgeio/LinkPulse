package main

import (
	"context"
	"errors"
	"linkpulse/internal/config"
	"linkpulse/internal/server"
	"linkpulse/internal/store/postgres"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	// Fail fast: no database URL, no point starting the server
	if cfg.DatabaseURL == "" {
		slog.Error("DATABASE_URL is required - set in backend/.env")
		os.Exit(1)
	}

	logger := setupLogger(cfg)
	slog.SetDefault(logger)

	// Connect to PostgreSQL and verify it answers a ping
	pool, err := postgres.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		slog.Error("cannot connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	slog.Info("database connected")

	srv := &http.Server{
		Addr: 				":" + cfg.AppPort,
		Handler: 			server.NewRouter(logger, pool),
		ReadTimeout: 	10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout: 	60 * time.Second,
	}

	go func() {
		slog.Info("Server Started", "addr", srv.Addr, "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Block until ctrl + c / terminal Signal arrives.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<- stop

	// graceful shutdown: finish in-flight request before exiting
	// This pattern will later flush the click buffer too
	slog.Info("shutting down server gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}
	slog.Info("Server stopped.")
}

// setupLogger builds a slog.Logger from configuration
func setupLogger(cfg config.Config) *slog.Logger {
	var level slog.Level
	switch strings.ToLower(cfg.LogLevel) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if strings.ToLower(cfg.LogFormat) == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}
