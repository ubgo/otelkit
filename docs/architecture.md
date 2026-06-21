# Architecture

otelkit is a *bootstrap*, not an SDK. It wires the official `go.opentelemetry.io/otel` SDK; it never reimplements spans, meters, or the log data model. Its value is correctness, ergonomics, and neutrality.

## The pipeline

```
Config (preset < options < env / file)
        │
        ▼
   Resource  ──────────────┐  (service.name + version + env + detectors)
        │                  │
        ▼                  ▼
  Exporters          Providers
  (per signal)   TracerProvider  ── batch/sync ── span exporter
   http/stdout   MeterProvider   ── periodic   ── metric exporter
   /grpc(contrib) LoggerProvider ── batch       ── log exporter
        │                  │
        ▼                  ▼
                   *Telemetry handle
        ┌──────────┼─────────────┬───────────────┐
        ▼          ▼             ▼               ▼
   SetGlobal   Tracer()    ForceFlush()      Shutdown()
 (otel globals) accessors                (logs→metrics→traces)
```

## Components

1. **Config resolution.** `defaultConfig()` carries spec defaults. A `Preset` fills in vendor data, `With*` options layer on top, then the `OTEL_*` environment overlays (unless `WithEnvOverrides(false)`). If `OTEL_CONFIG_FILE` is set, otelkit delegates the whole pipeline to `otelconf` and ignores the rest (file wins per spec).

2. **Resource.** Built from `service.name`/`version`, `deployment.environment.name`, the detector token set (process/os/host by default), and any extra attributes. `service.name` falls back to `unknown_service:<binary>` and can never be silently dropped.

3. **Exporters.** Per signal, per transport. The core ships OTLP/HTTP and stdout; OTLP/gRPC lives in `contrib/otelkit-grpc` and registers itself via `RegisterGRPC`. Exporter construction is pure — no global side effects.

4. **Providers.** A `TracerProvider` (chosen sampler, batch or sync processor), a `MeterProvider` (periodic reader, selectable temporality), and a `LoggerProvider` (batch processor). A disabled signal yields a no-op provider, never nil.

5. **Handle.** `*Telemetry` owns all three providers and the propagator. `SetGlobal()` registers them; `ForceFlush`/`Shutdown` operate across all three; `RunOnSignal` is the SIGTERM helper.

## The endpoint/path rules

otelkit owns all port and path construction so the classic OTLP footguns can't happen. Given a host, `host:port`, or URL and a transport:

- **gRPC** → bare `host:port`, default port `4317`, never a path.
- **HTTP, generic endpoint** → ensure scheme + port (default `4318`), then append `/v1/<signal>` to the existing base path, guarding against a double-append.
- **HTTP, per-signal endpoint** (`EndpointIsURL`) → used verbatim, no append (mirrors the OTLP spec).
- **stdout** → no endpoint.

This is implemented as pure string logic and is exhaustively unit-tested (every footgun case).

## Shutdown ordering

`Shutdown` flushes logs, then metrics, then traces — so a prior phase's errors still reach the log/trace collectors before those providers close. All three are attempted regardless of individual failures; errors aggregate via `errors.Join`. `RunOnSignal` runs `Shutdown` on a fresh background context so a cancelled application context can't abort the flush.

## Module split

| Module | Contains | Heavy deps |
|---|---|---|
| `github.com/ubgo/otelkit` | core: config, resource, providers, presets, diagnostics, HTTP + stdout exporters, otelconf delegation | none beyond `otel/*` + `otelconf` + `autoexport` |
| `github.com/ubgo/otelkit/contrib/otelkit-grpc` | OTLP/gRPC exporters | `google.golang.org/grpc` |

Keeping gRPC in contrib means the core's dependency graph stays free of `grpc` for the (common) HTTP-only deployment.

## Relationship to other libraries

otelkit owns the OTEL **pipeline**. It does not write log lines — that's a logger's job. `github.com/ubgo/logger`'s OTEL sink consumes the `LoggerProvider` otelkit builds (the only shared seam is the OTEL `log.LoggerProvider` type; neither module imports the other). For graceful multi-resource shutdown (HTTP + DB + queues + telemetry, phased), see `github.com/ubgo/shutdown`.
