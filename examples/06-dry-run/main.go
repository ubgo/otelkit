// Example 06 — dry run: print the resolved effective config, export nothing.
//
//	go run ./06-dry-run
//
// WithDryRun prints the fully-resolved configuration (auth headers redacted) and
// routes telemetry to stdout instead of the configured backend — so you can
// verify exactly what otelkit decided without touching a collector.
package main

import (
	"context"
	"log"

	"github.com/ubgo/otelkit"
)

func main() {
	ctx := context.Background()

	tel, err := otelkit.Init(ctx,
		otelkit.WithService("dry-run-example", "1.0.0"),
		otelkit.WithEnvironment("staging"),
		otelkit.WithPreset(otelkit.PresetGrafanaCloud("123456", "secret-token", "https://otlp-gateway-prod-eu-west-2.grafana.net/otlp")),
		otelkit.WithDryRun(),
	)
	if err != nil {
		log.Fatalf("otelkit init: %v", err)
	}
	tel.SetGlobal()
	defer tel.Shutdown(ctx)

	// The effective config (endpoint per signal, redacted headers, sampler,
	// temporality, TLS) was printed to stderr during Init. Telemetry below goes
	// to stdout, not Grafana.
	_, span := tel.Tracer("dry-run-example").Start(ctx, "hello")
	span.End()
}
