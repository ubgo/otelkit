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
