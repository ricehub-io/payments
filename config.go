package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port         string
	PolarSandbox bool
	PolarToken   string
}

// NewConfig loads .env file and parses it into new config struct.
// Returns error if file could not be loaded.
// Exits if any required environment variable is missing or invalid.
func NewConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("godotenv load: %w", err)
	}

	sandboxStr := getOptEnv("POLAR_SANDBOX", "false")
	sandbox, err := strconv.ParseBool(sandboxStr)
	if err != nil {
		log.Fatalf("could not parse '%s' as bool: %v", sandboxStr, err)
	}

	return &Config{
		Port:         getOptEnv("PORT", "50051"),
		PolarSandbox: sandbox,
		PolarToken:   getEnv("POLAR_TOKEN"),
	}, nil
}

// getEnv fetches given environment variable, exiting if it's not set.
//
// Use it to get environment variables that are required and can't have a default value.
func getEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		log.Fatalf("required environment variable %s is not set", key)
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
