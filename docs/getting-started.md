# Getting started

`otelkit` turns OpenTelemetry on in a Go service with one constructor. This guide takes you from zero to correlated traces, metrics, and logs reaching a backend.

## Install

```bash
go get github.com/ubgo/otelkit
```

gRPC support is a separate module (keeps `google.golang.org/grpc` out of the core):

```bash
go get github.com/ubgo/otelkit/contrib/otelkit-grpc
```

## The three-line setup

```go
tel, err := otelkit.Init(ctx, otelkit.WithService("checkout", "1.4.2"), otelkit.WithPreset(otelkit.PresetStdout()))
if err != nil { log.Fatal(err) }
tel.SetGlobal()
defer tel.Shutdown(ctx)
```

- **`Init`** resolves config (preset < options < env), builds the enabled providers, and returns one handle. It does no global registration on its own.
- **`SetGlobal`** registers the providers + propagator on the OpenTelemetry globals, so `otel.Tracer(...)`, `otel.Meter(...)`, and any OTEL-aware library pick them up.
- **`Shutdown`** flushes and shuts down all three signals in order (logs → metrics → traces), aggregating errors. Call it once on exit.

After `SetGlobal`, create spans the normal OTEL way:

```go
ctx, span := otel.Tracer("checkout").Start(ctx, "charge")
defer span.End()
```

## Point at a real backend

Swap the preset. otelkit owns the endpoint, port, path, auth header, and metric temporality for each vendor:

```go
otelkit.WithPreset(otelkit.PresetHyperDX("<ingestion-key>", "")) // or PresetGrafanaCloud / PresetHoneycomb / PresetDatadog / PresetNewRelic / PresetCollector
```

See [presets.md](./presets.md) for the full matrix.

## Long-running services

For a server, replace the `defer` with the signal helper, which blocks until SIGTERM/SIGINT and then flushes:

```go
go server.ListenAndServe()
if err := tel.RunOnSignal(ctx); err != nil {
	log.Printf("telemetry shutdown errors: %v", err)
}
```

## Fail loud, not silent

The OTEL SDK drops exports silently. Turn that into a startup error:

```go
tel, err := otelkit.Init(ctx,
	otelkit.WithPreset(otelkit.PresetCollector("collector:4318", otelkit.TransportHTTP)),
	otelkit.WithSelfTest(), // sends one span synchronously; errors if unreachable
)
```

See [diagnostics.md](./diagnostics.md).

## Next

- [configuration.md](./configuration.md) — the full `OTEL_*` environment surface and option precedence.
- [presets.md](./presets.md) — vendor presets and what each one encodes.
- [diagnostics.md](./diagnostics.md) — self-test, connectivity probe, dry-run, error handler.
- [declarative-config.md](./declarative-config.md) — `OTEL_CONFIG_FILE` delegation to `otelconf`.
- [architecture.md](./architecture.md) — how the pieces fit.
- [../examples](../examples) — runnable programs.
