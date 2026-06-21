# Diagnostics — loud, not silent

OpenTelemetry's default failure mode is silence. An exporter with a wrong endpoint, auth key, protocol, or TLS setting quietly retries and drops data; your app keeps running and your dashboards stay empty with no error. otelkit turns those failures into specific, loud signals.

## Boot self-test

`WithSelfTest()` sends one span synchronously during `Init` and force-flushes it, surfacing the export error the async batch processor would otherwise hide.

```go
tel, err := otelkit.Init(ctx,
	otelkit.WithPreset(otelkit.PresetHyperDX(apiKey, "")),
	otelkit.WithSelfTest(),
)
if err != nil {
	// e.g. "self-test failed: ... 401 Unauthorized" — wrong key, caught at startup.
	log.Fatalf("telemetry misconfigured: %v", err)
}
```

It's opt-in (it sends a real span). It verifies the *first hop* — that the backend or collector accepts the data (auth + endpoint + protocol + TLS reachable). It does not guarantee the data is stored deep downstream, but that first hop is where the overwhelming majority of misconfiguration lives.

## Connectivity probe

`ProbeEndpoint` is the lighter-weight cousin — a TCP dial (and TLS handshake for TLS modes) with a precise diagnosis, no span sent:

```go
if err := otelkit.ProbeEndpoint(ctx, "collector:4318", otelkit.TransportHTTP, otelkit.TLSModeTLS); err != nil {
	// "cannot reach collector:4318 — check host/port/DNS …" or
	// "TLS handshake to collector:4318 failed — check the certificate …"
	log.Printf("collector unreachable: %v", err)
}
```

Use it in a readiness check or before a deploy. It turns an opaque gRPC `Unavailable` into "wrong port? protocol? DNS? TLS?".

## Dry run

`WithDryRun()` prints the fully-resolved effective configuration (auth header values redacted) to stderr and routes telemetry to stdout instead of exporting. Use it to confirm exactly what otelkit decided — which endpoint per signal, which setting won the precedence — without touching a backend.

```go
otelkit.Init(ctx, otelkit.WithPreset(otelkit.PresetGrafanaCloud(id, token, ep)), otelkit.WithDryRun())
// stderr: otelkit: DRY RUN — effective configuration (no export):
//           service=svc version=1.0 env=staging
//           sampler=parentbased_always_on ratio=1 temporality=cumulative tls=tls
//           traces: transport=http endpoint=https://…/otlp/v1/traces headers={Authorization=<redacted>}
//           ...
```

## Export-error handler

otelkit installs a global OTEL error handler by default that logs export failures to stderr at WARN level — so a backend that starts rejecting data in production is visible, not silent. Route it into your own logger:

```go
otelkit.WithErrorHandler(func(err error) { mylog.Warn("otel export failed", "err", err) })
```

## Putting it together

A robust production bootstrap: a preset for correctness, a self-test to fail fast at boot, and an error handler so ongoing failures surface.

```go
tel, err := otelkit.Init(ctx,
	otelkit.WithService("checkout", version),
	otelkit.WithPreset(otelkit.PresetDatadog(ddKey, "")),
	otelkit.WithSelfTest(),
	otelkit.WithErrorHandler(logExportError),
)
```
