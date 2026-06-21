# Test coverage

Both modules are held at **100% line coverage** with the race detector, enforced in CI (the build fails under 100%).

```bash
task cover            # core module, 100% gate
cd contrib/otelkit-grpc && go test -race -cover ./...
```

| Module | Coverage |
|---|---|
| `github.com/ubgo/otelkit` | 100.0% |
| `github.com/ubgo/otelkit/contrib/otelkit-grpc` | 100.0% |

## What's tested

- **Endpoint/path resolution** — every footgun case: gRPC host-only/with-port/with-scheme, HTTP scheme + port defaulting, the `/v1/<signal>` append, the Grafana `/otlp` base, double-append guard, per-signal verbatim URLs, and the error paths.
- **Environment resolution** — the full `OTEL_*` surface, per-signal overrides, precedence, invalid-value handling, and `OTEL_SDK_DISABLED`.
- **Resource** — detector token sets, the `unknown_service:<binary>` fallback, attribute precedence.
- **Providers + exporters** — stdout end-to-end, HTTP construction, the gRPC-not-linked path, sampler and temporality selection, batch tuning.
- **Lifecycle** — ordered shutdown, partial-failure aggregation, `ForceFlush`, `RunOnSignal`, the no-op handle.
- **Diagnostics** — the export-error handler, the connectivity probe (reachable / unreachable / TLS handshake), self-test success and failure, dry-run with header redaction.
- **Presets** — each preset's exact endpoint, auth header name + value, and temporality.
- **Declarative config** — file delegation, read/parse/build error paths, and success.
- **gRPC contrib** — every transport + TLS mode + temporality combination, registration, and the missing-endpoint paths.

Coverage of branches that only fire on real export errors (e.g. resource-build failure) is achieved via small, documented test seams rather than left uncovered.
