# 06 — dry run

Print the fully-resolved effective configuration (auth headers redacted) and export nothing.

## What it shows

- `WithDryRun()` — `Init` prints the resolved config to stderr and rewrites every signal to the stdout transport, so telemetry goes to your terminal instead of the configured backend.

## When you'd use this

- Debugging "which setting won?" when presets, options, and `OTEL_*` env vars all contribute (precedence: preset < options < env).
- Verifying a new preset or endpoint locally with no collector.
- Confirming, in CI or a pre-deploy check, that auth headers are present (and **redacted** in the output — secrets never print).

## Run

```bash
go run ./06-dry-run
```

Expected stderr:

```
otelkit: DRY RUN — effective configuration (no export):
  service=dry-run-example version=1.0.0 env=staging
  sampler=parentbased_always_on ratio=1 temporality=cumulative tls=tls
  traces: transport=http endpoint=https://otlp-gateway-prod-eu-west-2.grafana.net/otlp/v1/traces headers={Authorization=<redacted>}
  metrics: ...
  logs: ...
```

Note the endpoint shows otelkit's resolved `/v1/<signal>` path and the redacted auth header — exactly what would be sent, minus the secret.
