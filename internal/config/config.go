package config

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	Environment        string `env:"ENVIRONMENT, default=dev"`
	Port               uint16 `env:"GRPC_PORT, default=50051"`
	WebhookPort        uint16 `env:"WEBHOOK_PORT, default=8080"`
	DatabaseURL        string `env:"DATABASE_URL, required"`
	SentryDSN          string `env:"SENTRY_DSN"`
	PolarSandbox       bool   `env:"POLAR_SANDBOX, default=false"`
	PolarToken         string `env:"POLAR_TOKEN, required"`
	PolarWebhookSecret string `env:"POLAR_WEBHOOK_SECRET, required"`
}

// NewConfig tries to load .env file in current working directory and
// parse it into new config struct.
// Returns error if file or variable could not be parsed.
func NewConfig(ctx context.Context) (*Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("loading env file: %w", err)
	}

	if env := os.Getenv("ENVIRONMENT"); env == "" {
		doppEnv := os.Getenv("DOPPLER_ENVIRONMENT")
		if err := os.Setenv("ENVIRONMENT", doppEnv); err != nil {
			return nil, fmt.Errorf("setting ENVIRONMENT variable: %w", err)
		}
	}

	var c Config
	if err := envconfig.Process(ctx, &c); err != nil {
		return nil, fmt.Errorf("processing config: %w", err)
	}

	return &c, nil
}
