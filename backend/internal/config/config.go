package config

import "os"

type Config struct {
	AppEnv					string
	AppPort					string
	AppBaseURL			string
	FrontendOrigin	string
	LogLevel				string
	LogFormat				string
}

func Load() Config {
	return Config{
		AppEnv: 				getEnv("APP_ENV", "development"),
		AppPort: 				getEnv("APP_PORT", "8080"),
		AppBaseURL:			getEnv("APP_BASE_URL", "http://localhost:8080") ,
		FrontendOrigin: getEnv("FRONTEND_ORIGIN", "http://localhost:3000"),
		LogLevel: 			getEnv("LOG_LEVEL", "info"),
		LogFormat: 			getEnv("LOG_FORMAT", "text"),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
