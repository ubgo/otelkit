# otelkit-grpc

[![Go Reference](https://pkg.go.dev/badge/github.com/ubgo/otelkit/contrib/otelkit-grpc.svg)](https://pkg.go.dev/github.com/ubgo/otelkit/contrib/otelkit-grpc) ![coverage](https://img.shields.io/badge/coverage-100%25-brightgreen) [![License](https://img.shields.io/badge/license-Apache--2.0-blue)](../../LICENSE)

**OTLP/gRPC exporter support for [`github.com/ubgo/otelkit`](../../).**

The core `otelkit` module ships OTLP/**HTTP** and **stdout** only, deliberately keeping `google.golang.org/grpc` (a large dependency) out of its graph. This module adds the gRPC exporters as a self-registering plug-in: import it and `TransportGRPC` — plus any gRPC-preferring preset — just works.

## Why it's a separate module

Most services export over OTLP/HTTP, which survives proxies and firewalls and needs no extra dependency. Pulling `google.golang.org/grpc` into every `otelkit` user just to satisfy the minority who need gRPC would be wrong. So gRPC lives here. The cost is one extra import line for gRPC users; the benefit is a lean core for everyone else.

If you select `TransportGRPC` **without** importing this module, `otelkit.Init` returns `otelkit.ErrGRPCNotLinked` — a clear, loud error at startup rather than a silent fallback.

## Install

```bash
go get github.com/ubgo/otelkit/contrib/otelkit-grpc
```

## Usage

A **blank import** is all you need — registration happens in the package's `init`:

```go
import (
	"github.com/ubgo/otelkit"
	_ "github.com/ubgo/otelkit/contrib/otelkit-grpc" // registers OTLP/gRPC exporters
)

tel, err := otelkit.Init(ctx,
	otelkit.WithService("svc", "1.0.0"),
	otelkit.WithPreset(otelkit.PresetCollector("otel-collector:4317", otelkit.TransportGRPC)),
)
```

If you ever reset the registration (`otelkit.RegisterGRPC(nil, nil, nil)`), call `otelkitgrpc.Register()` to wire it back.

## How it works

This module implements the three exporter-factory seams the core exposes (`otelkit.SpanExporterFactory`, `MetricExporterFactory`, `LogExporterFactory`) using the official OTLP/gRPC exporters, and registers them via `otelkit.RegisterGRPC`. For each signal it:

- resolves the endpoint to a bare `host:port` (default OTLP gRPC port `4317`) via `SignalConfig.GRPCTarget`,
- forwards configured headers,
- maps the `otelkit.TLSMode` to the gRPC transport credentials (`WithInsecure` for plaintext, `InsecureSkipVerify` creds for skip-verify, system roots otherwise),
- applies delta temporality for metrics when the config (or a preset like Datadog) asks for it.

## Configuration

gRPC honors the same `otelkit` options and presets as HTTP — `WithPreset`, `WithTLS`, `WithProtocol(otelkit.TransportGRPC)`, headers, and the `OTEL_EXPORTER_OTLP_*` environment variables. See the [core docs](../../docs/configuration.md).

## Example

A runnable program lives at [`../../examples/08-grpc`](../../examples/08-grpc).

## License

Apache-2.0. See [LICENSE](../../LICENSE).
