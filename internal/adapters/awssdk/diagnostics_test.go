package awssdk

import (
	"testing"

	"github.com/aws/smithy-go/logging"
)

func TestDiagnosticLoggerCountsKnownResponseCloseWarning(t *testing.T) {
	diagnostics := &Diagnostics{}
	logger := newDiagnosticLogger(diagnostics)

	logger.Logf(logging.Warn, responseCloseWarning)

	if got := diagnostics.ResponseCloseWarnings(); got != 1 {
		t.Fatalf("ResponseCloseWarnings() = %d, want 1", got)
	}
}
