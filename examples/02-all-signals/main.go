// Example 02 — all signals to stdout: traces, metrics, and logs.
//
//	go run ./02-all-signals
//
// PresetStdout enables all three signals on the stdout exporter, so you can see
// the full telemetry pipeline locally with no backend.
package main

import (
	"context"
	"log"
	"time"

	"github.com/ubgo/otelkit"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func main() {
	ctx := context.Background()

	tel, err := otelkit.Init(ctx,
		otelkit.WithService("all-signals", "1.0.0"),
		otelkit.WithEnvironment("dev"),
		otelkit.WithPreset(otelkit.PresetStdout()),
	)
	if err != nil {
		log.Fatalf("otelkit init: %v", err)
	}
	tel.SetGlobal()
	defer tel.Shutdown(ctx)

	// A trace.
	tctx, span := tel.Tracer("all-signals").Start(ctx, "work")
	time.Sleep(10 * time.Millisecond)
	span.SetAttributes(attribute.String("step", "demo"))
	span.End()

	// A metric.
	counter, _ := tel.MeterProvider().Meter("all-signals").Int64Counter("demo.runs")
	counter.Add(tctx, 1, metric.WithAttributes(attribute.String("kind", "example")))

	// A log record (through the OTEL log bridge).
	lg := tel.LoggerProvider().Logger("all-signals")
	_ = lg // emit via your logger of choice (e.g. github.com/ubgo/logger)

	log.Println("emitted trace + metric + log; flushing on shutdown")
}
