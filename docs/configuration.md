# Configuration

otelkit accepts configuration from three routes, with a defined precedence, and honors the standard OpenTelemetry environment variables.

## Precedence

```
programmatic defaults  <  preset  <  With* options  <  OTEL_* environment
```

Env wins by default (spec behavior). To make your own config system (options, or a config file like PKL) authoritative over raw `OTEL_*` vars, set:

```go
otelkit.WithEnvOverrides(false)
```

When `OTEL_CONFIG_FILE` is set, it overrides everything — see [declarative-config.md](./declarative-config.md).

## Options

| Option | Purpose |
|---|---|
| `WithService(name, version)` | `service.name` (required) + `service.version`. |
| `WithEnvironment(env)` | `deployment.environment.name` (e.g. `prod`). |
| `WithPreset(p)` | Apply a vendor preset (endpoint/auth/temporality/path). |
| `WithProtocol(t)` | Set transport (`TransportStdout`/`HTTP`/`GRPC`) for all signals. |
| `WithConfig(c)` | Supply a fully-formed `Config` (e.g. mapped from PKL). |
| `WithSampler(s)` / `WithSamplerRatio(r)` | Head sampler + ratio. |
| `WithMetricTemporality(t)` / `WithMetricInterval(d)` | Metric aggregation + reader interval. |
| `WithTLS(mode)` | `TLSModeTLS` / `TLSModePlaintext` / `TLSModeSkipVerify`. |
| `WithResourceDetectors(tokens)` | Token list: `env,host,os,process,container` / `all` / `none`. |
| `WithResourceAttrs(attrs...)` | Extra resource attributes (merged over detected). |
| `WithSelfTest()` / `WithDryRun()` | Loud diagnostics ([diagnostics.md](./diagnostics.md)). |
| `WithErrorHandler(fn)` | Route export errors into your logger (default: stderr). |
| `WithEnvOverrides(enabled)` | Whether `OTEL_*` env overrides options. |

## Environment variables

otelkit honors the spec `OTEL_*` surface. Defaults match the OpenTelemetry specification.

### General

| Variable | Default | Notes |
|---|---|---|
| `OTEL_SERVICE_NAME` | — | Sets `service.name`; precedence over the resource attr. Unset → `unknown_service:<binary>`. |
| `OTEL_RESOURCE_ATTRIBUTES` | — | `key=val,key=val` (W3C Baggage), merged into the resource. |
| `OTEL_SDK_DISABLED` | `false` | `true` → fully no-op providers (never nil). |

### OTLP exporter

Every variable below has per-signal overrides: `_TRACES_`, `_METRICS_`, `_LOGS_`.

| Variable | Default | Notes |
|---|---|---|
| `OTEL_EXPORTER_OTLP_PROTOCOL` | `http/protobuf` | `grpc` / `http/protobuf` / `http/json`. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `http://localhost:4318` (HTTP) / `:4317` (gRPC) | otelkit owns the `/v1/<signal>` path: it appends to the generic endpoint, uses a per-signal endpoint verbatim, and never appends for gRPC. |
| `OTEL_EXPORTER_OTLP_HEADERS` | — | `key=val,key=val` (auth, dataset, …). |
| `OTEL_EXPORTER_OTLP_INSECURE` | `false` | `true` → plaintext (no TLS). |

### Sampling, propagation, metrics

| Variable | Default | Notes |
|---|---|---|
| `OTEL_TRACES_SAMPLER` | `parentbased_always_on` | `always_on` / `always_off` / `traceidratio` / `parentbased_*`. |
| `OTEL_TRACES_SAMPLER_ARG` | `1.0` | Ratio in `[0,1]` for the ratio samplers. |
| `OTEL_PROPAGATORS` | `tracecontext,baggage` | Composite; `none` disables. |
| `OTEL_METRIC_EXPORT_INTERVAL` | `60000` (ms) | Periodic reader interval. |
| `OTEL_EXPORTER_OTLP_METRICS_TEMPORALITY_PREFERENCE` | `cumulative` | `delta` for Datadog/New Relic. |
| `OTEL_CONFIG_FILE` | — | Path to a declarative YAML config (delegates to `otelconf`). |

## The path footgun, handled

The single most common OTLP misconfiguration is the endpoint/port/path combination — gRPC on `:4318`, a missing or doubled `/v1/traces`, a generic-vs-per-signal endpoint mismatch. otelkit owns all of that: you give it a host, `host:port`, or URL and pick a transport; it derives the rest. See [architecture.md](./architecture.md) for the exact rules.
