# Vendor presets

A preset encodes one backend's ingest details as data: endpoint, transport, auth-header *name and format*, path quirk, and metric temporality. Switching backends is a one-line change, and the parts that silently break elsewhere are correct by construction.

```go
otelkit.Init(ctx, otelkit.WithService("svc", "1.0"), otelkit.WithPreset(otelkit.PresetHyperDX(apiKey, "")))
```

## Matrix

| Preset | Endpoint (blank → default) | Auth header | Auth value | Temporality |
|---|---|---|---|---|
| `PresetStdout()` | — | — | — | — |
| `PresetHyperDX(key, endpoint)` | `https://in-otel.hyperdx.io` | `authorization` | raw key (no `Bearer`) | default |
| `PresetGrafanaCloud(instanceID, token, endpoint)` | your `/otlp` gateway | `Authorization` | `Basic <base64(instanceID:token)>` | default |
| `PresetHoneycomb(key, dataset, endpoint)` | `https://api.honeycomb.io` | `x-honeycomb-team` | raw key | default; metrics also send `x-honeycomb-dataset` |
| `PresetDatadog(key, endpoint)` | `https://otlp.datadoghq.com` | `dd-api-key` | raw key | **delta (forced)** |
| `PresetNewRelic(key, endpoint)` | `https://otlp.nr-data.net` | `api-key` | license key | **delta** |
| `PresetCollector(endpoint, transport)` | required | — | — | configurable |

## Why presets matter

These are the traps a preset removes:

- **The `/otlp` base + `/v1/<signal>` append** (Grafana Cloud, Honeycomb): the SDK appends the signal path to a generic endpoint but uses a per-signal endpoint verbatim. Point at the wrong one and you get `…/v1/traces/v1/traces` or a 404 — both silent. The preset gets it right.
- **Delta vs cumulative temporality**: Datadog's direct OTLP intake *rejects* cumulative metrics (the default everywhere). `PresetDatadog` forces delta so metrics actually arrive; New Relic prefers it too.
- **Auth header name**: there is no `Bearer` consensus — HyperDX uses a raw `authorization`, Honeycomb `x-honeycomb-team`, New Relic `api-key`, Datadog `dd-api-key`. A wrong header name is a 401/403 or a silent drop.
- **Metrics-only headers**: Honeycomb metrics need a dataset header that traces/logs don't. The preset sets it only on the metrics signal.

## Overriding a preset

Options applied after `WithPreset` win (precedence: preset < options < env). Override anything:

```go
otelkit.Init(ctx,
	otelkit.WithPreset(otelkit.PresetHoneycomb(key, "metrics", "")),
	otelkit.WithSampler(otelkit.SamplerParentBasedTraceIDRatio),
	otelkit.WithSamplerRatio(0.1), // sample 10% of root traces
)
```

## The escape hatch

`PresetCollector(endpoint, transport)` is the vendor-neutral default: no auth, your endpoint, HTTP or gRPC. Use it for a local or in-cluster OTLP collector, then let the collector fan out to vendors.

```go
otelkit.WithPreset(otelkit.PresetCollector("otel-collector:4317", otelkit.TransportGRPC))
```

(gRPC requires importing `contrib/otelkit-grpc` — see the [README](../README.md#grpc).)
