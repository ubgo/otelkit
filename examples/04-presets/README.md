# 04 — vendor presets

Switch observability backends by changing one line.

## What it shows

- Selecting a `Preset` per backend (HyperDX, Grafana Cloud, Honeycomb, Datadog, New Relic) driven by an env var.
- That the *only* thing that changes between backends is the preset — the rest of the bootstrap is identical.

## When you'd use this

Whenever you might change vendors, run different backends per environment (e.g. stdout in dev, Grafana in staging, Datadog in prod), or just want the vendor's endpoint/auth/temporality quirks handled for you instead of hand-assembling them.

## Run

```bash
OTELKIT_BACKEND=stdout    go run ./04-presets   # default
OTELKIT_BACKEND=hyperdx   HYPERDX_API_KEY=...   go run ./04-presets
OTELKIT_BACKEND=grafana   GRAFANA_INSTANCE_ID=... GRAFANA_TOKEN=... GRAFANA_OTLP_ENDPOINT=https://otlp-gateway-<zone>.grafana.net/otlp go run ./04-presets
OTELKIT_BACKEND=honeycomb HONEYCOMB_API_KEY=...  go run ./04-presets
OTELKIT_BACKEND=datadog   DD_API_KEY=...         go run ./04-presets
OTELKIT_BACKEND=newrelic  NEW_RELIC_LICENSE_KEY=... go run ./04-presets
```

## What each preset encodes

Each preset sets the endpoint, the correct **auth header name and format** (no `Bearer` consensus across vendors), the path quirk, and metric **temporality** — e.g. `PresetDatadog` forces *delta* because Datadog's direct OTLP intake rejects cumulative. See [docs/presets.md](../../docs/presets.md) for the full matrix and the reasoning.
