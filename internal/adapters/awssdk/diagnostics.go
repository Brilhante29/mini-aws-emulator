package awssdk

import (
	"os"
	"sync/atomic"

	"github.com/aws/smithy-go/logging"
)

const responseCloseWarning = "failed to close HTTP response body, this may affect connection reuse"

type Diagnostics struct {
	responseCloseWarnings atomic.Uint64
}

func (d *Diagnostics) ResponseCloseWarnings() uint64 {
	return d.responseCloseWarnings.Load()
}

type diagnosticLogger struct {
	diagnostics *Diagnostics
	fallback    logging.Logger
}

func newDiagnosticLogger(diagnostics *Diagnostics) logging.Logger {
	return diagnosticLogger{
		diagnostics: diagnostics,
		fallback:    logging.NewStandardLogger(os.Stderr),
	}
}

func (l diagnosticLogger) Logf(classification logging.Classification, format string, values ...interface{}) {
	if classification == logging.Warn && format == responseCloseWarning {
		l.diagnostics.responseCloseWarnings.Add(1)
		return
	}
	l.fallback.Logf(classification, format, values...)
}
