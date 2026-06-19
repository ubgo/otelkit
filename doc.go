// Package otelkit is an OpenTelemetry SDK bootstrap for Go.
//
// otelkit stands up the OTEL trace/metric/log pipeline — providers, exporters,
// resource, propagators, and an ordered shutdown — from one explicit
// constructor, so an application points at a backend (HyperDX, Grafana,
// Honeycomb, Datadog, New Relic, a collector, or stdout) and gets correct,
// shutdown-safe telemetry without re-writing the usual ~150 lines of fiddly,
// failure-silent setup.
//
// otelkit is a bootstrap, not an SDK: it never reimplements spans, meters, or
// the log data model — it wires the official go.opentelemetry.io/otel SDK.
// Writing log lines is a logger's job (see github.com/ubgo/logger, which
// consumes the LoggerProvider this package builds). Auto-instrumentation lives
// in the OTEL contrib ecosystem.
//
// # Design goals
//
//   - Spec-compliant: honors the standard OTEL_* environment variables.
//   - Loud, not silent: opt-in boot self-test, connectivity probe, an error
//     handler that surfaces export failures, and a dry-run mode — because the
//     SDK's default failure mode is silence.
//   - Vendor presets as data: switch backends by changing one line.
//   - One handle, one ordered Shutdown; a real no-op on OTEL_SDK_DISABLED.
//   - Zero application dependencies.
//
// # Quick start
//
//	tel, err := otelkit.Init(ctx,
//	    otelkit.WithService("checkout", "1.4.2"),
//	    otelkit.WithPreset(otelkit.PresetCollector("localhost:4318", otelkit.TransportHTTP)),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	tel.SetGlobal()
//	defer tel.Shutdown(context.Background())
package otelkit
