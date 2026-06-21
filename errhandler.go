// errhandler.go — installs the global OTEL error handler (loud by default).
//
// The OTEL SDK reports asynchronous export failures to otel.SetErrorHandler,
// which is a no-op unless set. installErrorHandler wires a default stderr handler
// (overridable via WithErrorHandler) so a backend that starts rejecting data is
// visible instead of silently dropped. Called from Init for the non-no-op path.

package otelkit

import (
	"fmt"
	"os"

	"go.opentelemetry.io/otel"
)

// installErrorHandler registers a global OTEL error handler so that export
// failures — which the SDK otherwise swallows — are surfaced loudly. The
// default handler writes to stderr; override it with WithErrorHandler.
func installErrorHandler(c Config) {
	h := c.errorHandler
	if h == nil {
		h = func(err error) {
			fmt.Fprintf(os.Stderr, "otelkit: telemetry export error: %v\n", err)
		}
	}
	otel.SetErrorHandler(otel.ErrorHandlerFunc(h))
}
