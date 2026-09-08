// config - Loads application settings from environment variables with sensible defaults.
package config

import "os"

type Config struct {
	JWTSecret string
	Port      string
	DBPath    string
}

// Load reads configuration from environment variables.
func Load() *Config {
	return &Config{
		JWTSecret: getEnv("JWT_SECRET", "default-secret-change-me-in-production"),
		Port:      getEnv("PORT", "8080"),
		DBPath:    getEnv("DB_PATH", "tickets.db"),
	}
}

// getEnv returns the value of an environment variable, or the fallback if not set.
func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
