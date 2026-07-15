package report

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"
)

type Check struct {
	Service    string  `json:"service"`
	Name       string  `json:"name"`
	Passed     bool    `json:"passed"`
	DurationMS float64 `json:"duration_ms"`
	Error      string  `json:"error,omitempty"`
}

type Summary struct {
	PassedChecks             float64 `json:"passed_checks"`
	TotalChecks              float64 `json:"total_checks"`
	ConformanceRatePercent   float64 `json:"conformance_rate_percent"`
	P95OperationLatencyMS    float64 `json:"p95_operation_latency_ms"`
	OperationsPerSecond      float64 `json:"operations_per_second"`
	MeasuredOperations       float64 `json:"measured_operations"`
	FailedOperations         float64 `json:"failed_operations"`
	StartupMS                float64 `json:"startup_ms"`
	CoveragePercent          float64 `json:"coverage_percent"`
	SDKResponseCloseWarnings float64 `json:"sdk_response_close_warnings"`
}

type Result struct {
	Project     string            `json:"project"`
	Metric      string            `json:"metric"`
	Value       float64           `json:"value"`
	Unit        string            `json:"unit"`
	Timestamp   string            `json:"timestamp"`
	Command     string            `json:"command"`
	Repeat      int               `json:"repeat"`
	Summary     Summary           `json:"summary"`
	Environment map[string]string `json:"environment"`
	Services    []string          `json:"services"`
	Checks      []Check           `json:"checks"`
}

func P95(samples []time.Duration) float64 {
	if len(samples) == 0 {
		return 0
	}
	ordered := append([]time.Duration(nil), samples...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	index := int(float64(len(ordered))*0.95 + 0.999999)
	if index < 1 {
		index = 1
	}
	if index > len(ordered) {
		index = len(ordered)
	}
	return round3(float64(ordered[index-1].Microseconds()) / 1000)
}

func Round3(value float64) float64 { return round3(value) }

func Write(result Result, outputPath string) error {
	payload, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal result: %w", err)
	}
	payload = append(payload, '\n')
	if outputPath != "" {
		if err := os.WriteFile(outputPath, payload, 0o600); err != nil {
			return fmt.Errorf("write result: %w", err)
		}
	}
	_, err = os.Stdout.Write(payload)
	return err
}

func round3(value float64) float64 {
	return float64(int64(value*1000+0.5)) / 1000
}
