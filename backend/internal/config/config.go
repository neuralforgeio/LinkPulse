// Package config loads all application configuration from environment
// variables. Every value has a safe default so the server boots in
// development without any .env file — except DATABASE_URL and JWT_SECRET,
// which must be provided because there is no safe default for them.
package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all runtime configuration for the LinkPulse backend.
type Config struct {
    AppEnv                string
    AppPort               string
    AppBaseURL            string
    FrontendOrigin        string
    DatabaseURL           string
    JWTSecret             string
    JWTAccessTTL          time.Duration
    JWTRefreshTTL         time.Duration
    LogLevel              string
    LogFormat             string
    LogDir                string
    ClickSalt             string
    ClickBufferSize       int
    ClickFlushIntervalMS  int
    ClickFlushBatchSize   int
}

// Load reads configuration from the environment with development defaults.
func Load() Config {
    return Config{
        AppEnv:                getEnv("APP_ENV", "development"),
        AppPort:               getEnv("APP_PORT", "8080"),
        AppBaseURL:            getEnv("APP_BASE_URL", "http://localhost:8080"),
        FrontendOrigin:        getEnv("FRONTEND_ORIGIN", "http://localhost:3000"),
        DatabaseURL:           getEnv("DATABASE_URL", ""),
        JWTSecret:             getEnv("JWT_SECRET", ""),
        JWTAccessTTL:          getEnvDuration("JWT_ACCESS_TTL", 15*time.Minute),
        JWTRefreshTTL:         getEnvDuration("JWT_REFRESH_TTL", 30*24*time.Hour),
        LogLevel:              getEnv("LOG_LEVEL", "info"),
        LogFormat:             getEnv("LOG_FORMAT", "pretty"),
        LogDir:                getEnv("LOG_DIR", "logs"),
        ClickSalt:             getEnv("CLICK_SALT", ""),
        ClickBufferSize:       getEnvInt("CLICK_BUFFER_SIZE", 5000),
        ClickFlushIntervalMS:  getEnvInt("CLICK_FLUSH_INTERVAL_MS", 1000),
        ClickFlushBatchSize:   getEnvInt("CLICK_FLUSH_BATCH_SIZE", 500),
    }
}

// getEnv returns the environment value for key, or fallback if unset/empty.
func getEnv(key, fallback string) string {
    if v, ok := os.LookupEnv(key); ok && v != "" {
        return v
    }
    return fallback
}

// getEnvDuration parses a duration like "15m" or "720h", falling back to
// the default when unset or malformed.
func getEnvDuration(key string, fallback time.Duration) time.Duration {
    v, ok := os.LookupEnv(key)
    if !ok || v == "" {
        return fallback
    }
    d, err := time.ParseDuration(v)
    if err != nil {
        return fallback
    }
    return d
}

// getEnvInt parses an integer, falling back when unset or malformed.
func getEnvInt(key string, fallback int) int {
    v, ok := os.LookupEnv(key)
    if !ok || v == "" {
        return fallback
    }
    n, err := strconv.Atoi(v)
    if err != nil {
        return fallback
    }
    return n
}
