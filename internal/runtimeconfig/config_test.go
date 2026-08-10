package runtimeconfig

import "testing"

func TestFromEnvDefaultsToKumo(t *testing.T) {
	clearConfigEnv(t)
	cfg, err := FromEnv()
	if err != nil {
		t.Fatalf("FromEnv() error = %v", err)
	}
	if cfg.Provider != ProviderKumo {
		t.Fatalf("Provider = %q", cfg.Provider)
	}
	if cfg.Endpoint != "http://127.0.0.1:4566" {
		t.Fatalf("Endpoint = %q", cfg.Endpoint)
	}
	if cfg.Iterations != 25 {
		t.Fatalf("Iterations = %d", cfg.Iterations)
	}
	if cfg.Warmup != 5 {
		t.Fatalf("Warmup = %d", cfg.Warmup)
	}
}

func TestFromEnvGuardsRealAWS(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("CLOUD_PROVIDER", ProviderAWS)
	if _, err := FromEnv(); err == nil {
		t.Fatal("expected real AWS opt-in error")
	}
	t.Setenv("ALLOW_REAL_AWS", "true")
	if _, err := FromEnv(); err == nil {
		t.Fatal("expected explicit RUN_ID error")
	}
	t.Setenv("RUN_ID", "portfolio-123")
	if _, err := FromEnv(); err != nil {
		t.Fatalf("FromEnv() error = %v", err)
	}
}

func TestFromEnvRejectsUnsafeAndInvalidValues(t *testing.T) {
	clearConfigEnv(t)
	t.Setenv("RUN_ID", "Unsafe_ID")
	if _, err := FromEnv(); err == nil {
		t.Fatal("expected unsafe RUN_ID error")
	}
	t.Setenv("RUN_ID", "safe-id")
	t.Setenv("BENCHMARK_ITERATIONS", "zero")
	if _, err := FromEnv(); err == nil {
		t.Fatal("expected invalid iterations error")
	}
	t.Setenv("BENCHMARK_ITERATIONS", "25")
	t.Setenv("BENCHMARK_WARMUP_ITERATIONS", "0")
	if _, err := FromEnv(); err == nil {
		t.Fatal("expected invalid warmup error")
	}
}

func clearConfigEnv(t *testing.T) {
	for _, key := range []string{"CLOUD_PROVIDER", "CLOUD_ENDPOINT", "AWS_REGION", "RUN_ID", "BENCHMARK_WARMUP_ITERATIONS", "BENCHMARK_ITERATIONS", "SUITE_TIMEOUT", "REPEAT", "RESULT_PATH", "ALLOW_REAL_AWS"} {
		t.Setenv(key, "")
	}
}
