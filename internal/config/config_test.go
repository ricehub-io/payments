package config

import (
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

func TestNewConfig_Defaults(t *testing.T) {
	t.Chdir(t.TempDir())
	setRequiredEnv(t)

	cfg, err := NewConfig()
	require.NoError(t, err)

	assert.Equal(t, "dev", cfg.Environment)
	assert.Equal(t, uint16(50051), cfg.Port)
	assert.Equal(t, uint16(8080), cfg.WebhookPort)
	assert.Equal(t, "postgres://localhost/test", cfg.DatabaseURL)
	assert.Equal(t, "", cfg.SentryDSN)
	assert.False(t, cfg.PolarSandbox)
	assert.Equal(t, "test-token", cfg.PolarToken)
	assert.Equal(t, "test-secret", cfg.PolarWebhookSecret)
}

func TestNewConfig_ProdEnvironment(t *testing.T) {
	t.Chdir(t.TempDir())
	setRequiredEnv(t)
	t.Setenv("ENVIRONMENT", "prod")

	cfg, err := NewConfig()
	require.NoError(t, err)
	assert.Equal(t, "prod", cfg.Environment)
}

func TestNewConfig_InvalidEnvironment(t *testing.T) {
	t.Chdir(t.TempDir())
	setRequiredEnv(t)
	t.Setenv("ENVIRONMENT", "staging")

	_, err := NewConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ENVIRONMENT")
}

func TestNewConfig_MissingDatabaseURL(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("POLAR_TOKEN", "token")
	t.Setenv("POLAR_WEBHOOK_SECRET", "secret")

	_, err := NewConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DATABASE_URL")
}

func TestNewConfig_MissingPolarToken(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("POLAR_WEBHOOK_SECRET", "secret")

	_, err := NewConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "POLAR_TOKEN")
}

func TestNewConfig_MissingPolarWebhookSecret(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("POLAR_TOKEN", "token")

	_, err := NewConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "POLAR_WEBHOOK_SECRET")
}

func TestNewConfig_InvalidPortNonNumeric(t *testing.T) {
	t.Chdir(t.TempDir())
	setRequiredEnv(t)
	t.Setenv("PORT", "abc")

	_, err := NewConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "port")
}

func TestNewConfig_InvalidPortOutOfRange(t *testing.T) {
	t.Chdir(t.TempDir())
	setRequiredEnv(t)
	t.Setenv("PORT", "99999")

	_, err := NewConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "port")
}

func TestNewConfig_InvalidWebhookPort(t *testing.T) {
	t.Chdir(t.TempDir())
	setRequiredEnv(t)
	t.Setenv("WEBHOOK_PORT", "not-a-port")

	_, err := NewConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "webhook port")
}

func TestNewConfig_InvalidPolarSandbox(t *testing.T) {
	t.Chdir(t.TempDir())
	setRequiredEnv(t)
	t.Setenv("POLAR_SANDBOX", "maybe")

	_, err := NewConfig()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "polar sandbox")
}

func TestNewConfig_PolarSandboxTrue(t *testing.T) {
	t.Chdir(t.TempDir())
	setRequiredEnv(t)
	t.Setenv("POLAR_SANDBOX", "true")

	cfg, err := NewConfig()
	require.NoError(t, err)
	assert.True(t, cfg.PolarSandbox)
}

func TestNewConfig_SentryDSNOptional(t *testing.T) {
	t.Chdir(t.TempDir())
	setRequiredEnv(t)

	cfg, err := NewConfig()
	require.NoError(t, err)
	assert.Equal(t, "", cfg.SentryDSN)
}

func TestNewConfig_SentryDSNSet(t *testing.T) {
	t.Chdir(t.TempDir())
	setRequiredEnv(t)
	t.Setenv("SENTRY_DSN", "https://sentry.io/test")

	cfg, err := NewConfig()
	require.NoError(t, err)
	assert.Equal(t, "https://sentry.io/test", cfg.SentryDSN)
}

func TestNewConfig_CustomPorts(t *testing.T) {
	t.Chdir(t.TempDir())
	setRequiredEnv(t)
	t.Setenv("PORT", "9090")
	t.Setenv("WEBHOOK_PORT", "9091")

	cfg, err := NewConfig()
	require.NoError(t, err)
	assert.Equal(t, uint16(9090), cfg.Port)
	assert.Equal(t, uint16(9091), cfg.WebhookPort)
}
