package runtimeconfig

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"
)

const (
	ProviderKumo = "kumo"
	ProviderAWS  = "aws"
)

var safeRunID = regexp.MustCompile(`^[a-z0-9-]+$`)

type Config struct {
	Provider   string
	Endpoint   string
	Region     string
	RunID      string
	Iterations int
	Timeout    time.Duration
	Repeat     int
	OutputPath string
}

func FromEnv() (Config, error) {
	cfg := Config{
		Provider:   valueOrDefault("CLOUD_PROVIDER", ProviderKumo),
		Endpoint:   valueOrDefault("CLOUD_ENDPOINT", "http://127.0.0.1:4566"),
		Region:     valueOrDefault("AWS_REGION", "us-east-1"),
		RunID:      valueOrDefault("RUN_ID", "r1"),
		Iterations: intOrDefault("BENCHMARK_ITERATIONS", 25),
		Timeout:    durationOrDefault("SUITE_TIMEOUT", 60*time.Second),
		Repeat:     intOrDefault("REPEAT", 1),
		OutputPath: os.Getenv("RESULT_PATH"),
	}

	if cfg.Provider != ProviderKumo && cfg.Provider != ProviderAWS {
		return Config{}, fmt.Errorf("CLOUD_PROVIDER must be %q or %q", ProviderKumo, ProviderAWS)
	}
	if cfg.Provider == ProviderKumo && cfg.Endpoint == "" {
		return Config{}, fmt.Errorf("CLOUD_ENDPOINT is required for Kumo")
	}
	if cfg.Provider == ProviderAWS && os.Getenv("ALLOW_REAL_AWS") != "true" {
		return Config{}, fmt.Errorf("real AWS requires ALLOW_REAL_AWS=true")
	}
	if cfg.Provider == ProviderAWS && os.Getenv("RUN_ID") == "" {
		return Config{}, fmt.Errorf("real AWS requires an explicit globally unique RUN_ID")
	}
	if !safeRunID.MatchString(cfg.RunID) {
		return Config{}, fmt.Errorf("RUN_ID must contain only lowercase letters, numbers, and hyphens")
	}
	if cfg.Iterations < 1 || cfg.Iterations > 1000 {
		return Config{}, fmt.Errorf("BENCHMARK_ITERATIONS must be between 1 and 1000")
	}
	if cfg.Timeout <= 0 || cfg.Timeout > 15*time.Minute {
		return Config{}, fmt.Errorf("SUITE_TIMEOUT must be greater than zero and at most 15 minutes")
	}
	if cfg.Repeat < 0 {
		return Config{}, fmt.Errorf("REPEAT cannot be negative")
	}
	return cfg, nil
}

func valueOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func intOrDefault(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return -1
	}
	return parsed
}

func durationOrDefault(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return -1
	}
	return parsed
}
