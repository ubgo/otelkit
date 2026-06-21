# Declarative configuration

OpenTelemetry's declarative (file-based) configuration is stable. otelkit delegates to it so you can drive the whole pipeline from a YAML file instead of code.

## How it works

When `OTEL_CONFIG_FILE` (or the experimental `OTEL_EXPERIMENTAL_CONFIG_FILE`) is set, `Init` reads the file and delegates to the official [`otelconf`](https://pkg.go.dev/go.opentelemetry.io/contrib/otelconf) loader. Per the spec, **the file wins outright** — programmatic `With*` options and flat `OTEL_*` environment variables are ignored, except `${ENV}` substitution inside the YAML.

```go
// With OTEL_CONFIG_FILE set, this WithService is ignored.
tel, err := otelkit.Init(ctx, otelkit.WithService("ignored", "1.0"))
```

```bash
OTEL_CONFIG_FILE=./otel-config.yaml ./myservice
```

## Example file

```yaml
file_format: "0.3"

resource:
  attributes:
    - name: service.name
      value: checkout
    - name: service.version
      value: "1.4.2"
    - name: deployment.environment.name
      value: ${DEPLOY_ENV:-dev}

tracer_provider:
  processors:
    - batch:
        exporter:
          otlp_http:
            endpoint: ${OTLP_ENDPOINT:-http://localhost:4318}

meter_provider:
  readers:
    - periodic:
        exporter:
          otlp_http:
            endpoint: ${OTLP_ENDPOINT:-http://localhost:4318}

logger_provider:
  processors:
    - batch:
        exporter:
          otlp_http:
            endpoint: ${OTLP_ENDPOINT:-http://localhost:4318}
```

`${ENV}` and `${ENV:-default}` substitutions are resolved at load time.

## When to use which

- **Code + presets** (the default): best ergonomics, vendor presets, the loud diagnostics. Map your own config system (env, flags, PKL) into `With*` options.
- **Declarative file**: when ops wants a single portable OTEL config that's identical across languages, or to adopt the OTEL standard file format. otelkit consumes it without you writing a parser.

The two are mutually exclusive per invocation — if the file env var is set, the file is the source of truth.

## Note on otelkit features

The declarative path produces standard SDK providers via `otelconf`. otelkit's *extra* surface — vendor presets, the self-test, the dry-run print, the endpoint/path resolver — applies to the code path. When you delegate to a config file, you get exactly what the file describes.
