package conformance

import (
	"context"
	"testing"

	"github.com/Brilhante29/mini-aws-emulator/internal/testdouble"
)

func TestSuitePassesAllChecksWithSubstitutableAdapter(t *testing.T) {
	fake := testdouble.New()
	checks := New(fake.Ports(), "portfolio-test").Run(context.Background())
	if len(checks) != 18 {
		t.Fatalf("len(checks) = %d", len(checks))
	}
	for _, check := range checks {
		if !check.Passed {
			t.Errorf("check %s/%s failed: %s", check.Service, check.Name, check.Error)
		}
	}
}

func TestSuiteReportsAnAdapterFailureWithoutPanicking(t *testing.T) {
	fake := testdouble.New()
	fake.FailOperation = "put_object"
	checks := New(fake.Ports(), "portfolio-test").Run(context.Background())
	failed := 0
	for _, check := range checks {
		if !check.Passed {
			failed++
		}
	}
	if failed == 0 {
		t.Fatal("expected at least one failed check")
	}
}
