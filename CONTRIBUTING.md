# Contributing to ubgo/otelkit

Thanks for your interest in `ubgo/otelkit`. This repository is licensed under the **Apache License 2.0**. Pull requests are welcome.

## Workflow

1. Open an issue first for anything beyond a tiny fix. Discussing the design upfront avoids wasted work.
2. Fork + branch named after the issue: `fix/123-grafana-path`, `feat/456-honeycomb-preset`.
3. Run local checks: `task check` (gofmt + vet + the 100% coverage gate).
4. Use [Conventional Commits](https://www.conventionalcommits.org/) for the PR title.

## Repository layout

| Path | Module | Purpose |
|---|---|---|
| `.` | `github.com/ubgo/otelkit` | Core: config, resource, providers, presets, diagnostics, HTTP + stdout exporters, otelconf delegation. |
| `contrib/otelkit-grpc` | `github.com/ubgo/otelkit/contrib/otelkit-grpc` | OTLP/gRPC exporters. Keeps `google.golang.org/grpc` out of the core. |
| `examples/` | `github.com/ubgo/otelkit/examples` | Runnable example programs; must compile in CI. |
| `docs/`, `adr/`, `snippets/` | — | Documentation, decisions, copy-paste fragments. |

## Code conventions

- **Lean core dependencies.** The core depends only on stdlib + `go.opentelemetry.io/otel/*` + `contrib/otelconf` + `contrib/exporters/autoexport`. Anything heavier (notably `google.golang.org/grpc`) belongs in a `contrib/` module.
- **100% line coverage, race detector mandatory.** `task cover` fails under 100% on the core and the gRPC contrib; every test runs under `-race`. Cover error branches with small, documented test seams rather than leaving them uncovered.
- **It's a bootstrap, not an SDK.** Don't reimplement spans/meters/the log data model — wire the official SDK. Don't add a logging API (that's `github.com/ubgo/logger`) or auto-instrumentation.
- **No `init()` magic; pure constructors.** Globals are opt-in via `SetGlobal()`.
- **Loud, not silent.** New failure modes should surface as an error or a logged warning, never a silent drop.
- **New vendor presets** must encode the exact endpoint, auth-header name + format, path quirk, and temporality, with a test asserting each. Cite the vendor's OTLP doc in the PR.
- **Comments explain *why*, not *what*** — non-obvious invariants, spec rules, surprising tradeoffs.

## Testing locally

```sh
task test            # standard tests
task race            # with the race detector
task cover           # race + 100% coverage gate
task lint            # golangci-lint
task check           # fmt + vet + cover

# contrib + examples
cd contrib/otelkit-grpc && go test -race ./...
cd examples && go build ./...
```

## Public API stability

Until `v1.0.0`, the API may change between minor versions. After `v1.0.0`, breaking changes require a major version bump and a strong rationale.

## License of contributions

By submitting a pull request, you agree that your contribution is provided under the same Apache License 2.0 as the rest of the repository.
