# 01 — basic

The minimal otelkit setup: traces to a local OTLP/HTTP collector.

## What it shows

- `otelkit.Init` with `WithService` + `PresetCollector` over HTTP.
- `SetGlobal` so `otel.Tracer(...)` works anywhere, and `defer Shutdown` to flush on exit.
- Creating a single span.

## When you'd use this

Your starting point for any service exporting OTLP/HTTP to an in-cluster or local collector (the OpenTelemetry Collector, or a vendor agent listening on `:4318`). Swap `PresetCollector` for a vendor preset (see [04-presets](../04-presets)) to send straight to a SaaS backend.

## Run

```bash
go run ./01-basic
```

With a collector on `localhost:4318` the span arrives there. **With no collector running, the export fails silently** — which is the whole reason [05-self-test](../05-self-test) exists: add `WithSelfTest()` to turn that into a startup error.

## Key code

```go
tel, err := otelkit.Init(ctx,
	otelkit.WithService("basic-example", "1.0.0"),
	otelkit.WithPreset(otelkit.PresetCollector("localhost:4318", otelkit.TransportHTTP)),
	otelkit.WithTLS(otelkit.TLSModePlaintext),
)
tel.SetGlobal()
defer tel.Shutdown(ctx)
```
