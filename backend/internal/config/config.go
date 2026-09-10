package config

import (
	"os"
	"time"
)

// Config holds all runtime configuration for the LinkPulse backend.
type Config struct {
    AppEnv         string
    AppPort        string
    AppBaseURL     string
    FrontendOrigin string
    DatabaseURL    string
    JWTSecret      string
    JWTAccessTTL   time.Duration
    JWTRefreshTTL  time.Duration
    LogLevel       string
    LogFormat      string
    LogFile        string
}

// Load reads configuration from the environment with development defaults.
func Load() Config {
    return Config{
        AppEnv:         getEnv("APP_ENV", "development"),
        AppPort:        getEnv("APP_PORT", "8080"),
        AppBaseURL:     getEnv("APP_BASE_URL", "http://localhost:8080"),
        FrontendOrigin: getEnv("FRONTEND_ORIGIN", "http://localhost:3000"),
        DatabaseURL:    getEnv("DATABASE_URL", ""),
        JWTSecret:      getEnv("JWT_SECRET", ""),
        JWTAccessTTL:   getEnvDuration("JWT_ACCESS_TTL", 15*time.Minute),
        JWTRefreshTTL:  getEnvDuration("JWT_REFRESH_TTL", 30*24*time.Hour),
        LogLevel:       getEnv("LOG_LEVEL", "info"),
        LogFormat:      getEnv("LOG_FORMAT", "pretty"),
        LogFile:        getEnv("LOG_FILE", "logs/linkpulse.log"),
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
