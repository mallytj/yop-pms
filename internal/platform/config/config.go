package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port           string
	Environment    string // "dev", "prod"
	DatabaseURL    string
	RedisAddr      string
	RedisPassword  string
	AllowedOrigins []string
}

// MustLoad loads the configuration from environment variables.
// It panics if a required variable is missing.
func MustLoad() *Config {
	_ = godotenv.Load()

	origins := getEnv("ALLOWED_ORIGINS", "http://localhost:5173")

	cfg := &Config{
		Port:           getEnv("PORT", "8080"),
		Environment:    getEnv("APP_ENV", "dev"),
		DatabaseURL:    getRequiredEnv("DB_URL"),
		RedisAddr:      getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:  getEnv("REDIS_PASSWORD", ""),
		AllowedOrigins: strings.Split(origins, " "),
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getRequiredEnv(key string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		log.Fatalf("Fatal: environment variable %s is required but not set", key)
	}
	return value
}
