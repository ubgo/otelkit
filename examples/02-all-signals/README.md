# 02 — all signals

Traces, metrics, and logs — the full pipeline — to stdout, with no backend.

## What it shows

- `PresetStdout()` enables all three signals on the stdout exporter.
- Emitting one of each: a span, an `Int64Counter` increment, and access to the `LoggerProvider`.
- Reading the providers off the handle (`tel.MeterProvider()`, `tel.LoggerProvider()`).

## When you'd use this

Local development and debugging: see exactly what each signal looks like as it flows through the pipeline without standing up a collector. Also the simplest way to confirm metrics/logs wiring before pointing at a real backend.

## Run

```bash
go run ./02-all-signals
```

You'll see JSON span/metric/log records printed to stdout on shutdown.

## Note on logs

otelkit builds the `LoggerProvider` but does not write log lines itself — that's a logger's job. In a real app you'd hand `tel.LoggerProvider()` to a logger (e.g. [`github.com/ubgo/logger`](https://github.com/ubgo/logger)'s OTEL sink), which writes records through it and correlates them with the active trace.
