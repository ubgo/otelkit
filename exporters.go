// exporters.go — per-signal exporter construction and the gRPC registration seam.
//
// newSpanExporter / newMetricExporter / newLogExporter build OTLP/HTTP or stdout
// exporters from a SignalConfig + TLSMode (pure, no global side effects). gRPC is
// not implemented here: the core exposes SpanExporterFactory/MetricExporterFactory/
// LogExporterFactory and a RegisterGRPC hook that contrib/otelkit-grpc fills via
// its init; a TransportGRPC request with no registration returns ErrGRPCNotLinked.
// Consumed by providers.go.

package otelkit

import (
	"context"
	"crypto/tls"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// SpanExporterFactory builds an OTLP/gRPC span exporter. It is the seam the
// contrib/otelkit-grpc module registers; core never imports google.golang.org/grpc.
type SpanExporterFactory = func(ctx context.Context, sc SignalConfig, tlsMode TLSMode) (sdktrace.SpanExporter, error)

// MetricExporterFactory builds an OTLP/gRPC metric exporter.
type MetricExporterFactory = func(ctx context.Context, sc SignalConfig, tlsMode TLSMode, temp Temporality) (sdkmetric.Exporter, error)

// LogExporterFactory builds an OTLP/gRPC log exporter.
type LogExporterFactory = func(ctx context.Context, sc SignalConfig, tlsMode TLSMode) (sdklog.Exporter, error)

var (
	grpcSpanFactory   SpanExporterFactory
	grpcMetricFactory MetricExporterFactory
	grpcLogFactory    LogExporterFactory
)

// RegisterGRPC wires the OTLP/gRPC exporter factories. The contrib/otelkit-grpc
// module calls this from its package init, so applications enable gRPC simply
// by importing that module. Passing a nil factory leaves that signal's gRPC
// support unregistered.
func RegisterGRPC(span SpanExporterFactory, metric MetricExporterFactory, log LogExporterFactory) {
	grpcSpanFactory = span
	grpcMetricFactory = metric
	grpcLogFactory = log
}

// tlsClientOption converts the mode to the right HTTP exporter security setting.
// Returns nil when default TLS verification applies.
func tlsConfigForSkipVerify() *tls.Config {
	return &tls.Config{InsecureSkipVerify: true} //nolint:gosec // explicit opt-in via TLSModeSkipVerify
}

// newSpanExporter builds the span exporter for sc.
func newSpanExporter(ctx context.Context, sc SignalConfig, mode TLSMode) (sdktrace.SpanExporter, error) {
	switch sc.Transport {
	case TransportStdout:
		return stdouttrace.New()
	case TransportHTTP:
		url, err := sc.resolveEndpoint(SignalTraces, mode)
		if err != nil {
			return nil, err
		}
		opts := []otlptracehttp.Option{otlptracehttp.WithEndpointURL(url)}
		if len(sc.Headers) > 0 {
			opts = append(opts, otlptracehttp.WithHeaders(sc.Headers))
		}
		switch mode {
		case TLSModePlaintext:
			opts = append(opts, otlptracehttp.WithInsecure())
		case TLSModeSkipVerify:
			opts = append(opts, otlptracehttp.WithTLSClientConfig(tlsConfigForSkipVerify()))
		}
		return otlptracehttp.New(ctx, opts...)
	case TransportGRPC:
		if grpcSpanFactory == nil {
			return nil, ErrGRPCNotLinked
		}
		return grpcSpanFactory(ctx, sc, mode)
	default:
		return nil, ErrInvalidProtocol
	}
}

// newMetricExporter builds the metric exporter for sc, honoring temporality.
func newMetricExporter(ctx context.Context, sc SignalConfig, temp Temporality, mode TLSMode) (sdkmetric.Exporter, error) {
	switch sc.Transport {
	case TransportStdout:
		return stdoutmetric.New()
	case TransportHTTP:
		url, err := sc.resolveEndpoint(SignalMetrics, mode)
		if err != nil {
			return nil, err
		}
		opts := []otlpmetrichttp.Option{otlpmetrichttp.WithEndpointURL(url)}
		if len(sc.Headers) > 0 {
			opts = append(opts, otlpmetrichttp.WithHeaders(sc.Headers))
		}
		if temp == TemporalityDelta {
			opts = append(opts, otlpmetrichttp.WithTemporalitySelector(deltaTemporalitySelector))
		}
		switch mode {
		case TLSModePlaintext:
			opts = append(opts, otlpmetrichttp.WithInsecure())
		case TLSModeSkipVerify:
			opts = append(opts, otlpmetrichttp.WithTLSClientConfig(tlsConfigForSkipVerify()))
		}
		return otlpmetrichttp.New(ctx, opts...)
	case TransportGRPC:
		if grpcMetricFactory == nil {
			return nil, ErrGRPCNotLinked
		}
		return grpcMetricFactory(ctx, sc, mode, temp)
	default:
		return nil, ErrInvalidProtocol
	}
}

// newLogExporter builds the log exporter for sc.
func newLogExporter(ctx context.Context, sc SignalConfig, mode TLSMode) (sdklog.Exporter, error) {
	switch sc.Transport {
	case TransportStdout:
		return stdoutlog.New()
	case TransportHTTP:
		url, err := sc.resolveEndpoint(SignalLogs, mode)
		if err != nil {
			return nil, err
		}
		opts := []otlploghttp.Option{otlploghttp.WithEndpointURL(url)}
		if len(sc.Headers) > 0 {
			opts = append(opts, otlploghttp.WithHeaders(sc.Headers))
		}
		switch mode {
		case TLSModePlaintext:
			opts = append(opts, otlploghttp.WithInsecure())
		case TLSModeSkipVerify:
			opts = append(opts, otlploghttp.WithTLSClientConfig(tlsConfigForSkipVerify()))
		}
		return otlploghttp.New(ctx, opts...)
	case TransportGRPC:
		if grpcLogFactory == nil {
			return nil, ErrGRPCNotLinked
		}
		return grpcLogFactory(ctx, sc, mode)
	default:
		return nil, ErrInvalidProtocol
	}
}

// deltaTemporalitySelector reports delta temporality for every instrument kind
// — required by backends like Datadog direct intake.
func deltaTemporalitySelector(sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.DeltaTemporality
}
