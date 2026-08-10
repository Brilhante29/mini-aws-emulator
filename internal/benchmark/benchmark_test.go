package benchmark

import (
	"context"
	"testing"

	"github.com/Brilhante29/mini-aws-emulator/internal/testdouble"
)

func TestRunMeasuresNineOperationsPerIteration(t *testing.T) {
	fake := testdouble.New()
	result, err := Run(context.Background(), fake.Ports(), "portfolio-bench", 2, 3)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(result.Durations) != 27 {
		t.Fatalf("measured operations = %d", len(result.Durations))
	}
	if result.Failed != 0 {
		t.Fatalf("failed operations = %d", result.Failed)
	}
	if result.Elapsed <= 0 {
		t.Fatal("expected positive elapsed time")
	}
	if result.WarmupOperations != 18 {
		t.Fatalf("warmup operations = %d", result.WarmupOperations)
	}
}

func TestRunFailsWhenWarmupCannotComplete(t *testing.T) {
	fake := testdouble.New()
	fake.FailOperation = "put_object"
	if _, err := Run(context.Background(), fake.Ports(), "portfolio-bench", 1, 1); err == nil {
		t.Fatal("expected warmup failure")
	}
}
