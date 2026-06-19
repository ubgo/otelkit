// Package otelkitgrpc adds OTLP/gRPC exporter support to github.com/ubgo/otelkit.
//
// The core otelkit module ships HTTP + stdout only, keeping google.golang.org/grpc
// out of its dependency graph. Importing this module registers the gRPC exporter
// factories so that TransportGRPC (and gRPC-preferring presets) work:
//
//	import (
//	    "github.com/ubgo/otelkit"
//	    _ "github.com/ubgo/otelkit/contrib/otelkit-grpc" // enables gRPC
//	)
//
// The blank import is enough — registration happens in this package's init.
package otelkitgrpc

import (
	"context"
	"crypto/tls"

	"github.com/ubgo/otelkit"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc/credentials"
)

func init() { Register() }

// Register wires the OTLP/gRPC exporter factories into otelkit. It is called
// automatically from this package's init; call it explicitly only if you reset
// the registration with otelkit.RegisterGRPC(nil, nil, nil).
func Register() {
	otelkit.RegisterGRPC(newSpanExporter, newMetricExporter, newLogExporter)
}

// tlsCredsOption returns the gRPC transport-security dial option for the mode.
func newSpanExporter(ctx context.Context, sc otelkit.SignalConfig, mode otelkit.TLSMode) (sdktrace.SpanExporter, error) {
	target, err := sc.GRPCTarget()
	if err != nil {
		return nil, err
	}
	opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(target)}
	if len(sc.Headers) > 0 {
		opts = append(opts, otlptracegrpc.WithHeaders(sc.Headers))
	}
	switch mode {
	case otelkit.TLSModePlaintext:
		opts = append(opts, otlptracegrpc.WithInsecure())
	case otelkit.TLSModeSkipVerify:
		opts = append(opts, otlptracegrpc.WithTLSCredentials(skipVerifyCreds()))
	}
	return otlptracegrpc.New(ctx, opts...)
}

func newMetricExporter(ctx context.Context, sc otelkit.SignalConfig, mode otelkit.TLSMode, temp otelkit.Temporality) (sdkmetric.Exporter, error) {
	target, err := sc.GRPCTarget()
	if err != nil {
		return nil, err
	}
	opts := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(target)}
	if len(sc.Headers) > 0 {
		opts = append(opts, otlpmetricgrpc.WithHeaders(sc.Headers))
	}
	if temp == otelkit.TemporalityDelta {
		opts = append(opts, otlpmetricgrpc.WithTemporalitySelector(deltaSelector))
	}
	switch mode {
	case otelkit.TLSModePlaintext:
		opts = append(opts, otlpmetricgrpc.WithInsecure())
	case otelkit.TLSModeSkipVerify:
		opts = append(opts, otlpmetricgrpc.WithTLSCredentials(skipVerifyCreds()))
	}
	return otlpmetricgrpc.New(ctx, opts...)
}

func newLogExporter(ctx context.Context, sc otelkit.SignalConfig, mode otelkit.TLSMode) (sdklog.Exporter, error) {
	target, err := sc.GRPCTarget()
	if err != nil {
		return nil, err
	}
	opts := []otlploggrpc.Option{otlploggrpc.WithEndpoint(target)}
	if len(sc.Headers) > 0 {
		opts = append(opts, otlploggrpc.WithHeaders(sc.Headers))
	}
	switch mode {
	case otelkit.TLSModePlaintext:
		opts = append(opts, otlploggrpc.WithInsecure())
	case otelkit.TLSModeSkipVerify:
		opts = append(opts, otlploggrpc.WithTLSCredentials(skipVerifyCreds()))
	}
	return otlploggrpc.New(ctx, opts...)
}

// deltaSelector reports delta temporality for every instrument kind — required
// by backends like Datadog direct intake.
func deltaSelector(sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.DeltaTemporality
}

// skipVerifyCreds returns gRPC TLS credentials that skip server verification.
func skipVerifyCreds() credentials.TransportCredentials {
	return credentials.NewTLS(&tls.Config{InsecureSkipVerify: true}) //nolint:gosec // honors TLSModeSkipVerify
}
