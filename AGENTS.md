# AGENTS.md — codebase map for AI agents

Read this first. It is the orientation map for `ubgo/otelkit` so a fresh agent (or human) knows what every part does, how the pieces connect, and where to make a change — without reading every file.

## What this repo is

`otelkit` is an **OpenTelemetry bootstrap for Go**: one constructor (`Init`) that stands up the trace/metric/log SDK pipeline (providers + exporters + resource + propagators + ordered shutdown) and points it at a backend. It is a *bootstrap, not an SDK* — it wires `go.opentelemetry.io/otel`, never reimplements it. It does **not** write log lines (that's a logger's job) and does **not** do auto-instrumentation. See `README.md` for the user-facing pitch and `docs/architecture.md` for the design.

## Modules (multi-module repo)

| Path | Module | Role |
|---|---|---|
| `.` | `github.com/ubgo/otelkit` | Core: everything except gRPC. Deps: stdlib + `otel/*` + `otelconf` + `autoexport`. |
| `contrib/otelkit-grpc/` | `…/contrib/otelkit-grpc` | OTLP/gRPC exporters. Self-registers via `otelkit.RegisterGRPC` on blank import. Keeps `google.golang.org/grpc` out of the core. |
| `examples/` | `…/examples` | 8 runnable programs. Replaces the two above to `../`. Must compile in CI. |

`go.work` is a **local-dev convenience only and is gitignored** — CI builds each module standalone (each has its own `go.mod`, the examples/contrib use `replace` to the local core). Minimum Go: **1.25** (pulled up by `otelconf`).

## Core files — what each owns

| File | Responsibility |
|---|---|
| `doc.go` | Package overview godoc. |
| `types.go` | The enums: `Signal`, `Transport`, `Sampler`, `Temporality`, `TLSMode` + their OTEL-spec mappings (ports 4317/4318, `/v1/<signal>` suffixes, protocol/sampler value strings). |
| `errors.go` | Exported sentinel errors (`ErrMissingEndpoint`, `ErrGRPCNotLinked`, …). |
| `config.go` | `Config` + `SignalConfig` + defaults, **and** `resolveEndpoint`/`GRPCTarget` — the pure port+path resolver (the footgun-killer). |
| `env.go` | The `OTEL_*` environment overlay (`applyEnv`) + parse helpers. The only file that reads `os.Getenv`. |
| `options.go` | The public `With*` functional options + the `Option` type. |
| `resource.go` | Builds the OTEL `Resource` (detectors, `unknown_service` fallback, `deployment.environment.name`). |
| `exporters.go` | Per-signal OTLP/HTTP + stdout exporter construction; the gRPC factory seam + `RegisterGRPC`. |
| `providers.go` | Builds the Tracer/Meter/Logger providers (+ sampler, no-op providers for disabled signals). |
| `otelkit.go` | **The entry point**: `Init`, the `*Telemetry` handle, accessors, `SetGlobal`, `ForceFlush`, `SelfTest`, the propagator builder, and the dry-run printer. |
| `shutdown.go` | `Shutdown` (ordered logs→metrics→traces, `errors.Join`) + `RunOnSignal`. |
| `errhandler.go` | Installs the global OTEL error handler (loud-by-default). |
| `probe.go` | `ProbeEndpoint` — the connectivity diagnostic. |
| `presets.go` | Vendor presets (`PresetHyperDX`, …) + `Preset`/`WithPreset`. |
| `declarative.go` | `OTEL_CONFIG_FILE` delegation to `otelconf`. |

Every file starts with a header comment describing its role; every exported symbol has godoc.

## The three flows to understand

1. **Config resolution** (`otelkit.go:Init` → `options.go` → `env.go` / `declarative.go`): start from `defaultConfig()`, apply preset+options, then either delegate to a config file (`OTEL_CONFIG_FILE`) or overlay `OTEL_*` env. Precedence: **defaults < preset < options < env**; a config file wins outright.
2. **The Init pipeline** (`otelkit.go:Init`): resolve config → `buildResource` → `buildTracerProvider`/`buildMeterProvider`/`buildLoggerProvider` (each builds an exporter via `exporters.go`) → install error handler → assemble `*Telemetry` → optional self-test. `OTEL_SDK_DISABLED` short-circuits to a no-op handle.
3. **Endpoint/path resolution** (`config.go:resolveEndpoint`): the single place that turns a host/URL + transport into the correct dial target — gRPC `host:port` (4317), HTTP with the `/v1/<signal>` append rule (and double-append guard), per-signal URLs verbatim. This is the most-tested logic; if you touch OTLP endpoints, this is the file.

## Conventions (also in CONTRIBUTING.md)

- **100% line coverage, `-race`, enforced in CI.** `task cover` fails under 100%. Cover error branches with small test seams (e.g. `buildResourceFn`, `newSDKFn`) rather than leaving them uncovered.
- **Lean core deps.** Anything heavy (notably gRPC) goes in a `contrib/` module behind the factory seam.
- **No `init()` magic; pure constructors.** Globals are opt-in via `SetGlobal`.
- **Loud, not silent.** New failure modes surface as an error or logged warning.
- **Comments explain *why*, not *what*** — the names carry the *what*.

## Running things

```sh
task cover                                  # core: race + 100% gate
cd contrib/otelkit-grpc && go test -race ./...
cd examples && go build ./...
```

## Where to look for X

- "How do I configure it?" → `docs/configuration.md` (env vars) + `options.go`.
- "Add a vendor" → `presets.go` + `docs/presets.md`; assert the exact endpoint/auth/temporality in `presets_test.go`.
- "Why isn't data arriving?" → `probe.go`, `errhandler.go`, the dry-run in `otelkit.go`, and `docs/diagnostics.md`.
- "Add gRPC behavior" → `contrib/otelkit-grpc/grpc.go`.
- "The design rationale" → `docs/adr/` (Architecture Decision Records).
