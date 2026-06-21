# Migrating to otelkit

If you have a hand-rolled OpenTelemetry bootstrap — the familiar ~150 lines that build three providers, attach OTLP exporters, set globals, and wire shutdown — replace it with one `otelkit.Init`.

## Before (typical hand-rolled setup)

```go
res, _ := resource.New(ctx, resource.WithAttributes(semconv.ServiceName("checkout")))

traceExp, _ := otlptracehttp.New(ctx, otlptracehttp.WithEndpoint("collector:4318"), otlptracehttp.WithInsecure())
tp := sdktrace.NewTracerProvider(sdktrace.WithResource(res), sdktrace.WithBatcher(traceExp))

metricExp, _ := otlpmetrichttp.New(ctx, /* ... */)
mp := sdkmetric.NewMeterProvider(sdkmetric.WithResource(res), sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp)))

logExp, _ := otlploghttp.New(ctx, /* ... */)
lp := sdklog.NewLoggerProvider(sdklog.WithResource(res), sdklog.WithProcessor(sdklog.NewBatchProcessor(logExp)))

otel.SetTracerProvider(tp)
otel.SetMeterProvider(mp)
global.SetLoggerProvider(lp)
otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

shutdown := func(ctx context.Context) error {
	// ... flush + shut down each provider, hope the ordering is right ...
}
```

## After

```go
tel, err := otelkit.Init(ctx,
	otelkit.WithService("checkout", version),
	otelkit.WithPreset(otelkit.PresetCollector("collector:4318", otelkit.TransportHTTP)),
	otelkit.WithTLS(otelkit.TLSModePlaintext),
)
if err != nil { log.Fatal(err) }
tel.SetGlobal()
defer tel.Shutdown(ctx)
```

## What you gain

- **Vendor presets** — point at HyperDX/Grafana/Honeycomb/Datadog/New Relic by changing one line, with the auth header, path quirk, and temporality correct.
- **gRPC** — a blank import of `contrib/otelkit-grpc` instead of swapping every exporter package.
- **Loud diagnostics** — `WithSelfTest()`, `ProbeEndpoint`, `WithDryRun()`, and a default export-error handler, instead of silent drops.
- **The full `OTEL_*` env surface** — operators configure with the standard variables, including the per-signal endpoint/path rules handled for you.
- **A real no-op on `OTEL_SDK_DISABLED`** — the Go SDK doesn't honor it; otelkit returns no-op providers (never nil).
- **An ordered `Shutdown`** — logs → metrics → traces with `errors.Join`, plus a ready-made `RunOnSignal` SIGTERM helper.
- **Declarative config** — set `OTEL_CONFIG_FILE` and otelkit delegates to `otelconf`.

## Mapping your existing config

If your config comes from a struct (env, flags, a config system like PKL), build an `otelkit.Config` and pass it with `WithConfig`, then `WithEnvOverrides(false)` to keep your config authoritative:

```go
tel, _ := otelkit.Init(ctx,
	otelkit.WithConfig(otelkit.Config{
		ServiceName:    cfg.ServiceName,
		ServiceVersion: cfg.Version,
		Traces:         otelkit.SignalConfig{Enabled: cfg.Traces, Transport: otelkit.TransportHTTP, Endpoint: cfg.Endpoint, Headers: authHeaders},
		Metrics:        otelkit.SignalConfig{Enabled: cfg.Metrics, Transport: otelkit.TransportHTTP, Endpoint: cfg.Endpoint, Headers: authHeaders},
		Logs:           otelkit.SignalConfig{Enabled: cfg.Logs, Transport: otelkit.TransportHTTP, Endpoint: cfg.Endpoint, Headers: authHeaders},
	}),
	otelkit.WithEnvOverrides(false),
)
```
