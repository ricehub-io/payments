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
	Reflection   bool
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

	return &Config{
		Port:         getOptEnv("PORT", "50051"),
		Reflection:   getOptEnvBool("REFLECTION", "false"),
		PolarSandbox: getOptEnvBool("POLAR_SANDBOX", "false"),
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

func getOptEnvBool(key, fallback string) bool {
	valStr := getOptEnv(key, fallback)
	val, err := strconv.ParseBool(valStr)
	if err != nil {
		log.Fatalf("could not parse '%s' as bool: %v", valStr, err)
	}
	return val
}
