package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Environment        string
	Port               uint16
	WebhookPort        uint16
	DatabaseURL        string
	SentryDSN          string
	PolarSandbox       bool
	PolarToken         string
	PolarWebhookSecret string
}

// NewConfig loads .env file and parses it into new config struct.
// Returns error if file could not be loaded.
// Exits if any required environment variable is missing or invalid.
func NewConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("godotenv load: %w", err)
	}

	env := getOptEnv("ENVIRONMENT", "dev")
	if env != "dev" && env != "prod" {
		log.Fatalf("Invalid environment config value '%s', must be dev or prod!", env)
	}

	return &Config{
		Environment:        env,
		Port:               getOptEnvUint16("PORT", "50051"),
		WebhookPort:        getOptEnvUint16("WEBHOOK_PORT", "8080"),
		DatabaseURL:        getEnv("DATABASE_URL"),
		SentryDSN:          getOptEnv("SENTRY_DSN", ""),
		PolarSandbox:       getOptEnvBool("POLAR_SANDBOX", "false"),
		PolarToken:         getEnv("POLAR_TOKEN"),
		PolarWebhookSecret: getEnv("POLAR_WEBHOOK_SECRET"),
	}, nil
}

// getEnv fetches given environment variable, exiting if it's not set.
//
// Use it to get environment variables that are required and can't have a default value.
func getEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("Required config field '%s' is not set!", key)
	}
	return val
}

// getOptEnv fetches an environment variable defaulting to given value if not set.
func getOptEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getOptEnvBool(key, fallback string) bool {
	valStr := getOptEnv(key, fallback)
	val, err := strconv.ParseBool(valStr)
	if err != nil {
		log.Fatalf("Could not parse config value '%s' as bool: %v", valStr, err)
	}
	return val
}

func getOptEnvUint16(key, fallback string) uint16 {
	valStr := getOptEnv(key, fallback)
	val, err := strconv.ParseUint(valStr, 10, 16)
	if err != nil {
		log.Fatalf("Could not parse config value '%s' as uint16: %v", valStr, err)
	}
	return uint16(val)
}
