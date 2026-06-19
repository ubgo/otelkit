# ubgo/otelkit

**The boring, correct, loud way to turn OpenTelemetry on in a Go service.**

`otelkit` stands up the OpenTelemetry trace/metric/log pipeline — providers, exporters, resource, propagators, and an ordered shutdown — from one explicit constructor. Point it at a backend (HyperDX, Grafana Cloud, Honeycomb, Datadog, New Relic, an OTLP collector, or stdout) and get correct, shutdown-safe telemetry without re-writing the usual ~150 lines of fiddly, failure-silent setup.

It is a **bootstrap, not an SDK**: it wires the official `go.opentelemetry.io/otel` SDK rather than reimplementing it. Writing log lines is a logger's job — `otelkit` exposes the `LoggerProvider` that [`github.com/ubgo/logger`](https://github.com/ubgo/logger) (and any OTEL log bridge) consumes.

[![License](https://img.shields.io/badge/license-Apache--2.0-blue)](./LICENSE) ![Go](https://img.shields.io/badge/go-1.24-00ADD8?logo=go) ![coverage](https://img.shields.io/badge/coverage-100%25-brightgreen)

## Why otelkit

OpenTelemetry's Go SDK ships excellent primitives and **no opinionated bootstrap**, and its dominant failure mode is **silence**: a wrong port (4317 vs 4318), wrong protocol, a missing `/v1/<signal>` path, an unflushed batch on exit, or a cumulative-vs-delta mismatch all fail *without an error* — producing empty dashboards and no clue why. `otelkit` fixes that:

- **Spec-compliant** — honors the standard `OTEL_*` environment variables and defaults.
- **Vendor presets as data** — switch backends in one line; the preset encodes the endpoint, auth header name/format, path quirk, and metric temporality.
- **Loud, not silent** — an export-error handler, a connectivity probe, a dry-run mode, and an opt-in boot self-test turn silent misconfiguration into a specific startup error.
- **One knob, no footguns** — `otelkit` owns all port + `/v1/<signal>` path construction.
- **One handle, one ordered `Shutdown`** + a ready-made signal helper; a real no-op on `OTEL_SDK_DISABLED`.
- **Future-proof** — delegates to the now-stable declarative config (`otelconf`) when `OTEL_CONFIG_FILE` is set.
- **Zero application dependencies.**

## Install

```bash
go get github.com/ubgo/otelkit
```

gRPC support lives in a separate module so the core stays free of `google.golang.org/grpc`:

```bash
go get github.com/ubgo/otelkit/contrib/otelkit-grpc
```

## Quick start

```go
package main

import (
	"context"
	"log"

	"github.com/ubgo/otelkit"
)

func main() {
	ctx := context.Background()

	tel, err := otelkit.Init(ctx,
		otelkit.WithService("checkout", "1.4.2"),
		otelkit.WithEnvironment("prod"),
		otelkit.WithPreset(otelkit.PresetHyperDX("<ingestion-key>", "")),
	)
	if err != nil {
		log.Fatal(err)
	}
	tel.SetGlobal()
	defer tel.Shutdown(ctx)

	// ... run your service; create spans/metrics via the OTEL globals ...
}
```

For a long-running service, replace the `defer` with the signal helper:

```go
go runServer()
if err := tel.RunOnSignal(ctx); err != nil { // blocks until SIGTERM/SIGINT, then flushes
	log.Printf("shutdown errors: %v", err)
}
```

## Vendor presets

One named constructor per backend, encoding the parts that silently break:

| Preset | Auth header | Notes |
|---|---|---|
| `PresetStdout()` | — | Local dev; all signals to stdout. |
| `PresetHyperDX(key, endpoint)` | `authorization` (raw key, no `Bearer`) | Defaults to `https://in-otel.hyperdx.io`. |
| `PresetGrafanaCloud(instanceID, token, endpoint)` | `Authorization: Basic <b64>` | Endpoint is the `/otlp` base; `/v1/<signal>` is appended automatically. |
| `PresetHoneycomb(key, dataset, endpoint)` | `x-honeycomb-team` | Metrics additionally send `x-honeycomb-dataset`. |
| `PresetDatadog(key, endpoint)` | `dd-api-key` | **Forces delta temporality** (Datadog rejects cumulative). |
| `PresetNewRelic(key, endpoint)` | `api-key` | Prefers delta temporality. |
| `PresetCollector(endpoint, transport)` | — | Generic OTLP, no auth. The vendor-neutral escape hatch. |

Switching backend is a one-line change:

```go
otelkit.WithPreset(otelkit.PresetGrafanaCloud("123456", "<token>", "https://otlp-gateway-prod-eu-west-2.grafana.net/otlp"))
```

## Diagnostics — fail loud, at boot

```go
tel, err := otelkit.Init(ctx,
	otelkit.WithPreset(otelkit.PresetHoneycomb(key, "metrics", "")),
	otelkit.WithSelfTest(),                 // send one span synchronously; error if the backend is unreachable
	otelkit.WithErrorHandler(myLogger),     // route export failures into your logs (default: stderr)
)
```

- **`WithSelfTest()`** — sends one span through the real pipeline at startup and returns the export error the async batcher would otherwise hide.
- **Connectivity probe** — `otelkit.ProbeEndpoint(ctx, endpoint, transport, tlsMode)` diagnoses DNS / port / protocol / TLS problems with a human-readable message.
- **`WithDryRun()`** — prints the resolved effective config (auth headers redacted) and routes telemetry to stdout, so you can verify wiring with no backend.
- **Export-error handler** — installed by default (stderr); override with `WithErrorHandler`.

## gRPC

```go
import (
	"github.com/ubgo/otelkit"
	_ "github.com/ubgo/otelkit/contrib/otelkit-grpc" // blank import enables gRPC
)

tel, _ := otelkit.Init(ctx,
	otelkit.WithPreset(otelkit.PresetCollector("localhost:4317", otelkit.TransportGRPC)),
)
```

Without the contrib import, selecting `TransportGRPC` returns `otelkit.ErrGRPCNotLinked` — loud, not silent.

## Configuration sources

`otelkit` accepts config from three independent routes (precedence: preset < options < env):

1. **Programmatic** — `WithService`, `WithPreset`, `WithProtocol`, `WithSampler`, `WithTLS`, … (map your own config system, e.g. PKL, into these).
2. **`OTEL_*` environment variables** — the full standard surface (protocol, endpoint, headers, timeout, sampler, propagators, temporality). Set `WithEnvOverrides(false)` to make programmatic values authoritative.
3. **Declarative config file** — set `OTEL_CONFIG_FILE` and `otelkit` delegates to the stable [`otelconf`](https://pkg.go.dev/go.opentelemetry.io/contrib/otelconf) loader (file wins; flat env is ignored except `${ENV}` substitution).

## Migrating from a hand-rolled bootstrap

If you have a `lace/gotel`-style bootstrap (build three providers, attach HTTP exporters, set globals, shut down): replace it with one `otelkit.Init(...)` call plus the matching preset. `otelkit` adds gRPC, vendor presets, loud diagnostics, the `OTEL_*` surface, a real no-op on `OTEL_SDK_DISABLED`, and an ordered `Shutdown` that flushes all three signals (instead of returning on the first error).

## License

Apache-2.0. See [LICENSE](./LICENSE).
