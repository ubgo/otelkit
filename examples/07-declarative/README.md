# 07 — declarative config

Drive otelkit from an OpenTelemetry config **file** instead of code.

## What it shows

- Setting `OTEL_CONFIG_FILE` makes `otelkit.Init` delegate to the stable [`otelconf`](https://pkg.go.dev/go.opentelemetry.io/contrib/otelconf) loader.
- The file wins outright — the `WithService` option in the code is ignored.
- `${ENV}` / `${ENV:-default}` substitution inside the YAML.

## When you'd use this

When ops wants a single, portable OpenTelemetry configuration that's identical across languages and services, or to adopt the OTEL standard file format. otelkit consumes it without you writing a parser. For code-first ergonomics (presets, self-test, dry-run), use the option path instead — the two are mutually exclusive per invocation.

## Run

```bash
OTEL_CONFIG_FILE=./otel-config.yaml go run ./07-declarative

# override the substituted values:
OTEL_CONFIG_FILE=./otel-config.yaml DEPLOY_ENV=prod OTLP_ENDPOINT=http://collector:4318 go run ./07-declarative
```

Without `OTEL_CONFIG_FILE` set, the program falls back to the code path (the ignored `WithService`).

## The config file

[`otel-config.yaml`](./otel-config.yaml) follows the `opentelemetry-configuration` schema: `file_format`, a `resource` block, and a `tracer_provider` with a batch OTLP/HTTP exporter. See [docs/declarative-config.md](../../docs/declarative-config.md) for the full shape and the file-vs-env precedence rule.
