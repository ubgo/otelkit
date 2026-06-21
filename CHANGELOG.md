# Changelog

All notable changes to `ubgo/otelkit` are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/) and the project adheres to
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added
- Core bootstrap: `Init`, `*Telemetry` handle, accessors, `SetGlobal`, ordered
  `Shutdown` (logs → metrics → traces, `errors.Join`), `ForceFlush`, `RunOnSignal`.
- Full `OTEL_*` environment-variable surface with `defaults < file < env` precedence.
- Endpoint/path resolver that owns port (4317/4318) and `/v1/<signal>` construction.
- Resource detectors (process/os/host), `unknown_service:<binary>` fallback,
  `deployment.environment.name`.
- Vendor presets: HyperDX, Grafana Cloud, Honeycomb, Datadog (delta-forced),
  New Relic, Collector, Stdout.
- Loud diagnostics: export-error handler, connectivity probe, opt-in self-test,
  dry-run effective-config print.
- Real no-op on `OTEL_SDK_DISABLED`.
- Declarative-config delegation to `otelconf` via `OTEL_CONFIG_FILE`.
- `contrib/otelkit-grpc`: OTLP/gRPC exporters (keeps `grpc` out of the core).
- 100% line coverage (race) on the core and the gRPC contrib.

### Docs & examples
- Full `docs/` guide set: getting-started, configuration (env-var reference),
  presets (vendor matrix), diagnostics, declarative-config, migration, architecture.
- Runnable `examples/` (01-basic … 08-grpc), each built in CI.
- Architecture Decision Records under `adr/`, copy-paste `snippets/`, and `COVERAGE.md`.
