# Snippets

Copy-paste fragments for common tasks. For full runnable programs see [../../examples](../../examples).

## Minimal

```go
tel, _ := otelkit.Init(ctx, otelkit.WithService("svc", "1.0"), otelkit.WithPreset(otelkit.PresetStdout()))
tel.SetGlobal()
defer tel.Shutdown(ctx)
```

## Production, with a vendor + self-test

```go
tel, err := otelkit.Init(ctx,
	otelkit.WithService("checkout", version),
	otelkit.WithEnvironment("prod"),
	otelkit.WithPreset(otelkit.PresetDatadog(os.Getenv("DD_API_KEY"), "")), // forces delta metrics
	otelkit.WithSelfTest(),
)
if err != nil { log.Fatalf("telemetry: %v", err) }
tel.SetGlobal()
```

## Graceful shutdown on SIGTERM

```go
go server.ListenAndServe()
if err := tel.RunOnSignal(ctx); err != nil {
	log.Printf("telemetry shutdown errors: %v", err)
}
```

## Sample 10% of traces

```go
otelkit.Init(ctx,
	otelkit.WithSampler(otelkit.SamplerParentBasedTraceIDRatio),
	otelkit.WithSamplerRatio(0.10),
)
```

## Enable gRPC

```go
import _ "github.com/ubgo/otelkit/contrib/otelkit-grpc"

otelkit.WithPreset(otelkit.PresetCollector("collector:4317", otelkit.TransportGRPC))
```

## Skip TLS verification (dev/self-signed only)

```go
otelkit.WithTLS(otelkit.TLSModeSkipVerify)
```

## Verify reachability in a readiness check

```go
if err := otelkit.ProbeEndpoint(ctx, "collector:4318", otelkit.TransportHTTP, otelkit.TLSModeTLS); err != nil {
	return fmt.Errorf("collector unreachable: %w", err)
}
```

## Map a config struct (e.g. PKL) and keep it authoritative

```go
otelkit.Init(ctx,
	otelkit.WithConfig(otelkit.Config{
		ServiceName:    cfg.ServiceName,
		ServiceVersion: cfg.Version,
		Traces:         otelkit.SignalConfig{Enabled: true, Transport: otelkit.TransportHTTP, Endpoint: cfg.Endpoint, Headers: headers},
	}),
	otelkit.WithEnvOverrides(false), // options win over raw OTEL_* env
)
```

## Disable telemetry entirely

```bash
OTEL_SDK_DISABLED=true ./myservice   # Init returns no-op providers (never nil)
```
