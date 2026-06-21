# 05 — boot self-test

Fail loudly at startup when telemetry is misconfigured, instead of discovering empty dashboards hours later.

## What it shows

- `WithSelfTest()` — `Init` sends one span synchronously and returns the export error the async batch processor would otherwise hide.
- `otelkit.ProbeEndpoint(...)` — a standalone reachability check (TCP/TLS dial) with a precise diagnosis, no span sent.

## When you'd use this

Production services where silent telemetry loss is unacceptable. The OTEL SDK never surfaces export failures to your code — a wrong endpoint, API key, protocol, or TLS mode just drops data. The self-test converts that into a fatal startup error so a bad deploy fails fast.

## Run

```bash
go run ./05-self-test
```

With no collector on `localhost:4318` you'll get a fatal error like:

```
telemetry is misconfigured or unreachable: otelkit: self-test failed: ... connection refused
```

Start a collector (or point at a reachable one) and it proceeds.

## Self-test vs probe

- **`WithSelfTest()`** sends a real span — it verifies the *first hop* (auth + endpoint + protocol + TLS accepted). Opt-in because it emits one span.
- **`ProbeEndpoint`** only dials — cheaper, good for readiness checks, but doesn't validate auth.

Use the self-test at boot; use the probe in a recurring health check. See [docs/diagnostics.md](../../docs/diagnostics.md).
