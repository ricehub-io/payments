package config

import (
	"errors"
	"fmt"
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
// Returns error if file could not be loaded or parsed into config.
func NewConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("godotenv load: %w", err)
	}

	env := getOptEnv("ENVIRONMENT", "dev")
	if env != "dev" && env != "prod" {
		return nil, fmt.Errorf("'ENVIRONMENT' has invalid value '%s', must be 'dev' or 'prod'", env)
	}

	port, err := getOptEnvUint16("PORT", "50051")
	if err != nil {
		return nil, fmt.Errorf("port: %w", err)
	}

	whPort, err := getOptEnvUint16("WEBHOOK_PORT", "8080")
	if err != nil {
		return nil, fmt.Errorf("webhook port: %w", err)
	}

	dbURL, err := getEnv("DATABASE_URL")
	if err != nil {
		return nil, fmt.Errorf("database url: %w", err)
	}

	polarSandbox, err := getOptEnvBool("POLAR_SANDBOX", "false")
	if err != nil {
		return nil, fmt.Errorf("polar sandbox: %w", err)
	}

	polarToken, err := getEnv("POLAR_TOKEN")
	if err != nil {
		return nil, fmt.Errorf("polar token: %w", err)
	}

	polarWhSecret, err := getEnv("POLAR_WEBHOOK_SECRET")
	if err != nil {
		return nil, fmt.Errorf("polar webhook secret: %w", err)
	}

	return &Config{
		Environment:        env,
		Port:               port,
		WebhookPort:        whPort,
		DatabaseURL:        dbURL,
		SentryDSN:          getOptEnv("SENTRY_DSN", ""),
		PolarSandbox:       polarSandbox,
		PolarToken:         polarToken,
		PolarWebhookSecret: polarWhSecret,
	}, nil
}

// getEnv fetches given environment variable, exiting if it's not set.
//
// Use it to get environment variables that are required and can't have a default value.
func getEnv(key string) (string, error) {
	val := os.Getenv(key)
	if val == "" {
		return "", fmt.Errorf("required config field '%s' is not set", key)
	}
	return val, nil
}

// getOptEnv fetches an environment variable defaulting to given value if not set.
func getOptEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getOptEnvBool(key, fallback string) (bool, error) {
	valStr := getOptEnv(key, fallback)
	val, err := strconv.ParseBool(valStr)
	if err != nil {
		return false, fmt.Errorf("could not parse value '%s' as bool: %w", valStr, err)
	}
	return val, nil
}

func getOptEnvUint16(key, fallback string) (uint16, error) {
	valStr := getOptEnv(key, fallback)
	val, err := strconv.ParseUint(valStr, 10, 16)
	if err != nil {
		return 0, fmt.Errorf("could not parse value '%s' as uint16: %w", valStr, err)
	}
	return uint16(val), nil
}
