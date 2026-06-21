# 08 — OTLP/gRPC

Export over gRPC via the `contrib/otelkit-grpc` module.

## What it shows

- A **blank import** of `github.com/ubgo/otelkit/contrib/otelkit-grpc` registers the OTLP/gRPC exporter factories.
- `PresetCollector(endpoint, otelkit.TransportGRPC)` then sends over gRPC (default port `4317`).

## When you'd use this

When your collector or backend prefers OTLP/gRPC (some, like SigNoz Cloud or Dash0, default to it), or when you want gRPC's streaming/multiplexing. Most deployments are fine on HTTP — gRPC is opt-in precisely so HTTP-only users don't pay for `google.golang.org/grpc`.

## Run

```bash
go run ./08-grpc
```

Targets `localhost:4317`. With no collector, the export fails (loud, on flush) — start an OTLP/gRPC collector to see spans arrive.

## The important detail

Without the blank import, selecting `TransportGRPC` returns `otelkit.ErrGRPCNotLinked` at `Init` — a clear error, not a silent fallback:

```go
import _ "github.com/ubgo/otelkit/contrib/otelkit-grpc" // <- this line enables gRPC
```

See the [contrib module README](../../contrib/otelkit-grpc) for how registration works.
