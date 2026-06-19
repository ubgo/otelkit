package otelkit

import (
	"context"
	"fmt"
	"os"

	"go.opentelemetry.io/contrib/otelconf"
	"go.opentelemetry.io/otel/propagation"
)

// Test seams for the otelconf delegation. Indirected so the NewSDK error path
// is coverable without crafting a backend-specific invalid config.
var (
	parseYAMLFn = otelconf.ParseYAML
	newSDKFn    = otelconf.NewSDK
)

// configFilePath returns the declarative-config file path from the standard
// OTEL_CONFIG_FILE (preferred) or the experimental OTEL_EXPERIMENTAL_CONFIG_FILE
// that current Go SDKs still read. Empty when neither is set.
func configFilePath() string {
	if p := getenv("OTEL_CONFIG_FILE"); p != "" {
		return p
	}
	return getenv("OTEL_EXPERIMENTAL_CONFIG_FILE")
}

// initFromFile builds a Telemetry from a declarative YAML config via otelconf.
// Per the spec, when a config file is present it wins outright — flat OTEL_*
// env vars and programmatic options are ignored (only ${ENV} substitution
// inside the YAML applies, which otelconf handles).
func initFromFile(ctx context.Context, path string) (*Telemetry, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path comes from a trusted env var
	if err != nil {
		return nil, fmt.Errorf("otelkit: read config file %q: %w", path, err)
	}
	conf, err := parseYAMLFn(data)
	if err != nil {
		return nil, fmt.Errorf("otelkit: parse config file %q: %w", path, err)
	}
	sdk, err := newSDKFn(otelconf.WithContext(ctx), otelconf.WithOpenTelemetryConfiguration(*conf))
	if err != nil {
		return nil, fmt.Errorf("otelkit: build SDK from config %q: %w", path, err)
	}

	return &Telemetry{
		tp:         sdk.TracerProvider(),
		mp:         sdk.MeterProvider(),
		lp:         sdk.LoggerProvider(),
		propagator: propagatorOrDefault(sdk.Propagator()),
		shutdowns:  []shutdownFunc{sdk.Shutdown},
	}, nil
}

// propagatorOrDefault returns p, or the tracecontext+baggage default when p is
// nil (a declarative config may omit propagators).
func propagatorOrDefault(p propagation.TextMapPropagator) propagation.TextMapPropagator {
	if p == nil {
		return propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	}
	return p
}
