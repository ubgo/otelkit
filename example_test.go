package otelkit_test

import (
	"context"
	"log"

	"github.com/ubgo/otelkit"
)

// Minimal: stdout in local dev.
func ExampleInit() {
	ctx := context.Background()
	tel, err := otelkit.Init(ctx,
		otelkit.WithService("checkout", "1.4.2"),
		otelkit.WithPreset(otelkit.PresetStdout()),
		otelkit.WithEnvOverrides(false),
	)
	if err != nil {
		log.Fatal(err)
	}
	tel.SetGlobal()
	defer tel.Shutdown(ctx)
}

// Switch backends in one line via a preset.
func ExampleWithPreset() {
	ctx := context.Background()
	tel, err := otelkit.Init(ctx,
		otelkit.WithService("api", "2.0.0"),
		otelkit.WithEnvironment("prod"),
		otelkit.WithPreset(otelkit.PresetHyperDX("ingestion-key", "")),
	)
	if err != nil {
		log.Fatal(err)
	}
	tel.SetGlobal()
	defer tel.Shutdown(ctx)
}

// Catch a misconfigured backend at startup.
func ExampleWithSelfTest() {
	ctx := context.Background()
	tel, err := otelkit.Init(ctx,
		otelkit.WithPreset(otelkit.PresetCollector("localhost:4318", otelkit.TransportHTTP)),
		otelkit.WithSelfTest(),
	)
	if err != nil {
		log.Printf("telemetry self-test failed: %v", err)
		return
	}
	defer tel.Shutdown(ctx)
}

// Verify wiring with no backend.
func ExampleWithDryRun() {
	ctx := context.Background()
	tel, _ := otelkit.Init(ctx,
		otelkit.WithService("svc", "1"),
		otelkit.WithPreset(otelkit.PresetGrafanaCloud("123456", "token", "https://otlp-gateway.grafana.net/otlp")),
		otelkit.WithDryRun(),
		otelkit.WithEnvOverrides(false),
	)
	defer tel.Shutdown(ctx)
}

// Probe an endpoint's reachability with a precise diagnosis.
func ExampleProbeEndpoint() {
	ctx := context.Background()
	if err := otelkit.ProbeEndpoint(ctx, "localhost:4318", otelkit.TransportHTTP, otelkit.TLSModePlaintext); err != nil {
		log.Printf("collector unreachable: %v", err)
	}
}
