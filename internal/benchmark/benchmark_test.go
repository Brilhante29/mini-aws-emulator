package benchmark

import (
	"context"
	"testing"

	"github.com/Brilhante29/mini-aws-emulator/internal/testdouble"
)

func TestRunMeasuresNineOperationsPerIteration(t *testing.T) {
	fake := testdouble.New()
	result, err := Run(context.Background(), fake.Ports(), "portfolio-bench", 3)
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
}
