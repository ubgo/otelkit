# otelkit examples

Runnable programs, each in its own directory. From this folder:

```bash
go run ./01-basic
```

| # | Example | What it shows |
|---|---|---|
| 01 | [`01-basic`](./01-basic) | Traces to a local OTLP/HTTP collector — the minimal setup. |
| 02 | [`02-all-signals`](./02-all-signals) | Traces + metrics + logs to stdout (no backend needed). |
| 03 | [`03-k8s-prestop`](./03-k8s-prestop) | Graceful shutdown on SIGTERM with `RunOnSignal` (Kubernetes pod termination). |
| 04 | [`04-presets`](./04-presets) | Switch backends (HyperDX / Grafana / Honeycomb / Datadog / New Relic) in one line. |
| 05 | [`05-self-test`](./05-self-test) | Catch a misconfigured backend at startup with `WithSelfTest` + `ProbeEndpoint`. |
| 06 | [`06-dry-run`](./06-dry-run) | Print the resolved effective config (headers redacted), export nothing. |
| 07 | [`07-declarative`](./07-declarative) | Drive otelkit from an `OTEL_CONFIG_FILE` YAML (otelconf delegation). |
| 08 | [`08-grpc`](./08-grpc) | OTLP/gRPC via a blank import of `contrib/otelkit-grpc`. |

Every example compiles in CI (`go build ./...`). Most run offline; the ones that
target a collector (01, 03, 05, 08) print export errors to stderr when no
collector is reachable — exactly the loud-by-default behavior otelkit is about.
