package report

import (
	"testing"
	"time"
)

func TestP95UsesNearestRank(t *testing.T) {
	samples := make([]time.Duration, 100)
	for index := range samples {
		samples[index] = time.Duration(index+1) * time.Millisecond
	}
	if got := P95(samples); got != 95 {
		t.Fatalf("P95() = %v", got)
	}
}

func TestP95HandlesEmptyInput(t *testing.T) {
	if got := P95(nil); got != 0 {
		t.Fatalf("P95(nil) = %v", got)
	}
}
