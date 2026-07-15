package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Brilhante29/mini-aws-emulator/internal/adapters/awssdk"
	"github.com/Brilhante29/mini-aws-emulator/internal/benchmark"
	"github.com/Brilhante29/mini-aws-emulator/internal/conformance"
	"github.com/Brilhante29/mini-aws-emulator/internal/report"
	"github.com/Brilhante29/mini-aws-emulator/internal/runtimeconfig"
)

var (
	version       = "dev"
	kumoVersion   = "0.25.3"
	kumoDigest    = "sha256:7ea090ae0b6d1d34615e8b7bd04a2f1cd864ec640a6826a91e90f40e975e196b"
	awsSDKVersion = "1.41.9"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := runtimeconfig.FromEnv()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	startup := time.Duration(0)
	useLocalEndpoint := cfg.Provider == runtimeconfig.ProviderKumo
	if useLocalEndpoint {
		startup, err = waitForHealth(ctx, cfg.Endpoint)
		if err != nil {
			return err
		}
	}

	adapter, err := awssdk.New(ctx, awssdk.Options{
		Region:           cfg.Region,
		Endpoint:         cfg.Endpoint,
		UseLocalEndpoint: useLocalEndpoint,
		WaitForResources: cfg.Provider == runtimeconfig.ProviderAWS,
	})
	if err != nil {
		return err
	}

	prefix := "portfolio-aws-" + cfg.RunID
	checks := conformance.New(adapter.Ports(), prefix+"-check").Run(ctx)
	passed := 0
	for _, check := range checks {
		if check.Passed {
			passed++
		}
	}
	conformanceRate := 0.0
	if len(checks) > 0 {
		conformanceRate = report.Round3(float64(passed) * 100 / float64(len(checks)))
	}

	bench, err := benchmark.Run(ctx, adapter.Ports(), prefix+"-bench", cfg.Iterations)
	if err != nil {
		return err
	}
	measured := len(bench.Durations)
	succeeded := measured - bench.Failed
	operationsPerSecond := 0.0
	if bench.Elapsed > 0 {
		operationsPerSecond = report.Round3(float64(succeeded) / bench.Elapsed.Seconds())
	}

	providerVersion := "managed"
	providerDigest := "managed"
	if cfg.Provider == runtimeconfig.ProviderKumo {
		providerVersion = kumoVersion
		providerDigest = kumoDigest
	}

	result := report.Result{
		Project:   "mini-aws-emulator",
		Metric:    "conformance_rate_percent",
		Value:     conformanceRate,
		Unit:      "percent",
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Command:   "docker run --rm mini-aws-emulator",
		Repeat:    cfg.Repeat,
		Summary: report.Summary{
			PassedChecks:             float64(passed),
			TotalChecks:              float64(len(checks)),
			ConformanceRatePercent:   conformanceRate,
			P95OperationLatencyMS:    report.P95(bench.Durations),
			OperationsPerSecond:      operationsPerSecond,
			MeasuredOperations:       float64(measured),
			FailedOperations:         float64(bench.Failed),
			StartupMS:                report.Round3(float64(startup.Microseconds()) / 1000),
			CoveragePercent:          readCoverage(),
			SDKResponseCloseWarnings: float64(adapter.Diagnostics().ResponseCloseWarnings()),
		},
		Environment: map[string]string{
			"provider":         cfg.Provider,
			"provider_version": providerVersion,
			"provider_digest":  providerDigest,
			"runner_version":   version,
			"go":               runtime.Version(),
			"aws_sdk_go_v2":    awsSDKVersion,
			"region":           cfg.Region,
			"iterations":       strconv.Itoa(cfg.Iterations),
		},
		Services: []string{"s3", "sqs", "dynamodb"},
		Checks:   checks,
	}
	if err := report.Write(result, cfg.OutputPath); err != nil {
		return err
	}
	if passed != len(checks) || bench.Failed > 0 {
		return fmt.Errorf("conformance or benchmark failures: checks=%d/%d operations_failed=%d", passed, len(checks), bench.Failed)
	}
	return nil
}

func waitForHealth(ctx context.Context, endpoint string) (time.Duration, error) {
	started := time.Now()
	client := &http.Client{Timeout: time.Second}
	url := strings.TrimRight(endpoint, "/") + "/health"
	for {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return 0, err
		}
		response, err := client.Do(request)
		if err == nil {
			response.Body.Close()
			if response.StatusCode >= 200 && response.StatusCode < 300 {
				return time.Since(started), nil
			}
		}
		select {
		case <-ctx.Done():
			return 0, fmt.Errorf("wait for Kumo health: %w", ctx.Err())
		case <-time.After(25 * time.Millisecond):
		}
	}
}

func readCoverage() float64 {
	path := os.Getenv("COVERAGE_PATH")
	if path == "" {
		path = "/app/coverage-percent.txt"
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	value, err := strconv.ParseFloat(strings.TrimSpace(string(content)), 64)
	if err != nil {
		return 0
	}
	return report.Round3(value)
}
