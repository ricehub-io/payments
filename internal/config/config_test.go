package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("POLAR_TOKEN", "test-token")
	t.Setenv("POLAR_WEBHOOK_SECRET", "test-secret")
}

func TestNewConfig_HappyPath(t *testing.T) {
	t.Chdir(t.TempDir())
	setRequiredEnv(t)
	t.Setenv("ENVIRONMENT", "prod")
	t.Setenv("GRPC_PORT", "9090")
	t.Setenv("WEBHOOK_PORT", "9091")
	t.Setenv("POLAR_SANDBOX", "true")
	t.Setenv("SENTRY_DSN", "https://sentry.io/test")

	cfg, err := NewConfig(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "prod", cfg.Environment)
	assert.Equal(t, uint16(9090), cfg.Port)
	assert.Equal(t, uint16(9091), cfg.WebhookPort)
	assert.Equal(t, "postgres://localhost/test", cfg.DatabaseURL)
	assert.Equal(t, "https://sentry.io/test", cfg.SentryDSN)
	assert.True(t, cfg.PolarSandbox)
	assert.Equal(t, "test-token", cfg.PolarToken)
	assert.Equal(t, "test-secret", cfg.PolarWebhookSecret)
}

func TestNewConfig_DopplerEnvironmentFallback(t *testing.T) {
	t.Chdir(t.TempDir())
	setRequiredEnv(t)
	t.Setenv("DOPPLER_ENVIRONMENT", "prod")
	require.NoError(t, os.Unsetenv("ENVIRONMENT"))

	cfg, err := NewConfig(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "prod", cfg.Environment)
}

func TestNewConfig_ExplicitEnvironmentBeatsDoppler(t *testing.T) {
	t.Chdir(t.TempDir())
	setRequiredEnv(t)
	t.Setenv("ENVIRONMENT", "prod")
	t.Setenv("DOPPLER_ENVIRONMENT", "dev")

	cfg, err := NewConfig(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "prod", cfg.Environment)
}

func TestNewConfig_LoadsDotEnvFile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	require.NoError(t, os.Unsetenv("ENVIRONMENT"))
	require.NoError(t, os.Unsetenv("DOPPLER_ENVIRONMENT"))

	contents := "DATABASE_URL=postgres://from-dotenv/test\n" +
		"POLAR_TOKEN=dotenv-token\n" +
		"POLAR_WEBHOOK_SECRET=dotenv-secret\n" +
		"ENVIRONMENT=prod\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte(contents), 0o600))

	cfg, err := NewConfig(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "postgres://from-dotenv/test", cfg.DatabaseURL)
	assert.Equal(t, "dotenv-token", cfg.PolarToken)
	assert.Equal(t, "dotenv-secret", cfg.PolarWebhookSecret)
	assert.Equal(t, "prod", cfg.Environment)
}

func TestNewConfig_MissingDotEnvIsNotError(t *testing.T) {
	t.Chdir(t.TempDir())
	setRequiredEnv(t)

	_, err := NewConfig(context.Background())
	require.NoError(t, err)
}
