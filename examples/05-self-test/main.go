// Example 05 — boot self-test: fail loudly at startup, not in production.
//
//	go run ./05-self-test
//
// WithSelfTest sends one span synchronously during Init and returns the export
// error the async batch processor would otherwise hide. A wrong endpoint, auth
// key, protocol, or TLS mode surfaces here instead of as empty dashboards.
package main

import (
	"context"
	"log"

	"github.com/ubgo/otelkit"
)

func main() {
	ctx := context.Background()

	tel, err := otelkit.Init(ctx,
		otelkit.WithService("self-test-example", "1.0.0"),
		otelkit.WithPreset(otelkit.PresetCollector("localhost:4318", otelkit.TransportHTTP)),
		otelkit.WithTLS(otelkit.TLSModePlaintext),
		otelkit.WithSelfTest(),
	)
	if err != nil {
		// e.g. "self-test failed: ... connection refused" when no collector is up.
		log.Fatalf("telemetry is misconfigured or unreachable: %v", err)
	}
	tel.SetGlobal()
	defer tel.Shutdown(ctx)

	// A standalone probe is also available without sending a real span:
	if err := otelkit.ProbeEndpoint(ctx, "localhost:4318", otelkit.TransportHTTP, otelkit.TLSModePlaintext); err != nil {
		log.Printf("probe: %v", err)
	}

	log.Println("telemetry verified reachable at startup")
}
