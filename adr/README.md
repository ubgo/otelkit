# Architecture Decision Records

The decisions that shape `ubgo/otelkit`, each with its context and consequences.

---

## ADR 0001 — Bootstrap, not SDK

**Context.** OpenTelemetry's Go SDK provides excellent primitives but no opinionated assembly. Every service re-writes the same ~150 lines.

**Decision.** otelkit wires the official SDK; it never reimplements spans, meters, or the log data model. Its value is correctness + ergonomics + neutrality.

**Consequences.** We track the SDK's API and stay a thin layer. We do not compete with the SDK; we make it usable.

---

## ADR 0002 — Multi-module repository

**Context.** OTLP/gRPC pulls `google.golang.org/grpc`, a heavy dependency many deployments don't need.

**Decision.** The core module ships HTTP + stdout. OTLP/gRPC lives in `contrib/otelkit-grpc`, registered via `RegisterGRPC`. A blank import enables it.

**Consequences.** HTTP-only users get a small dependency graph. Selecting `TransportGRPC` without the contrib import returns `ErrGRPCNotLinked` — loud, not silent.

---

## ADR 0003 — Zero application dependencies

**Context.** The package it replaces depended on application code and validation libraries, so it only compiled inside its origin workspace.

**Decision.** Core dependencies are stdlib + `go.opentelemetry.io/otel/*` + `contrib/otelconf` + `contrib/exporters/autoexport`. Validation uses stdlib errors.

**Consequences.** Any project (or external user) can `go get` otelkit with no hidden coupling.

---

## ADR 0004 — Explicit construction, opt-in globals

**Context.** Hidden `init()` magic and constructors with global side effects make telemetry hard to test and embed.

**Decision.** `Init` builds providers and returns a handle with no global registration. `SetGlobal()` is opt-in. Constructors are pure.

**Consequences.** Two telemetry stacks can coexist in one process; tests don't fight global state.

---

## ADR 0005 — Own all endpoint/path construction

**Context.** The top OTLP misconfiguration is the endpoint/port/path combination: gRPC on 4318, a missing or doubled `/v1/<signal>`, generic-vs-per-signal mismatch — all silent.

**Decision.** otelkit derives port and path from one transport choice and an endpoint. Default OTLP protocol is `http/protobuf`.

**Consequences.** The footguns can't happen. The rules are pure logic, exhaustively unit-tested.

---

## ADR 0006 — Vendor presets as data

**Context.** Each backend has its own endpoint shape, auth header name/format, path quirk, and temporality preference. Getting one wrong is a silent failure.

**Decision.** A `Preset` is a function that fills a `Config` with a vendor's ingest details. Switching backends is one line.

**Consequences.** Correct-by-construction backend config; new backends are new presets, not core changes. `PresetCollector` is always available as the neutral escape hatch.

---

## ADR 0007 — Loud by default

**Context.** The SDK drops exports silently — the dominant real-world pain.

**Decision.** A default export-error handler (stderr), a connectivity probe, and a dry-run mode are built in. A boot self-test is opt-in (`WithSelfTest`) because it sends a real span.

**Consequences.** Misconfiguration surfaces as a startup error or a logged warning instead of empty dashboards.

---

## ADR 0008 — Real no-op on `OTEL_SDK_DISABLED`

**Context.** The Go SDK does not honor `OTEL_SDK_DISABLED`, and code that returns nil providers forces nil-checks everywhere.

**Decision.** When the variable is truthy, `Init` returns fully no-op providers (never nil).

**Consequences.** Telemetry can be turned off cleanly in CI/local with no call-site branching.

---

## ADR 0009 — Delegate to declarative config

**Context.** OpenTelemetry's file-based declarative configuration is now stable.

**Decision.** When `OTEL_CONFIG_FILE` is set, otelkit delegates to `otelconf`. The file wins; flat env and options are ignored except `${ENV}` substitution.

**Consequences.** Future-proof; we don't fight the standard. The code path keeps the preset/diagnostics ergonomics.

---

## ADR 0010 — Ordered, aggregating shutdown

**Context.** A naive shutdown that returns on the first provider error leaks the other two — lost telemetry on exit.

**Decision.** `Shutdown` flushes logs → metrics → traces, attempts all three, and aggregates errors with `errors.Join`. `RunOnSignal` shuts down on a fresh context.

**Consequences.** Telemetry isn't lost on exit when one provider errors; a cancelled app context can't abort the flush.

---

## ADR 0011 — License: Apache-2.0

**Context.** Ecosystem-wide license policy.

**Decision.** Apache-2.0, with `LICENSE` + `NOTICE`.

**Consequences.** Permissive reuse; original copyright preserved.
