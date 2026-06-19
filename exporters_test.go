package otelkit

import (
	"context"
	"errors"
	"testing"
	"time"

	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// --- fake gRPC exporters (the contrib seam) ---

type fakeSpanExporter struct{ exportErr error }

func (f fakeSpanExporter) ExportSpans(context.Context, []sdktrace.ReadOnlySpan) error {
	return f.exportErr
}
func (fakeSpanExporter) Shutdown(context.Context) error { return nil }

type fakeMetricExporter struct{}

func (fakeMetricExporter) Temporality(k sdkmetric.InstrumentKind) metricdata.Temporality {
	return sdkmetric.DefaultTemporalitySelector(k)
}
func (fakeMetricExporter) Aggregation(k sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return sdkmetric.DefaultAggregationSelector(k)
}
func (fakeMetricExporter) Export(context.Context, *metricdata.ResourceMetrics) error { return nil }
func (fakeMetricExporter) ForceFlush(context.Context) error                          { return nil }
func (fakeMetricExporter) Shutdown(context.Context) error                            { return nil }

type fakeLogExporter struct{}

func (fakeLogExporter) Export(context.Context, []sdklog.Record) error { return nil }
func (fakeLogExporter) Shutdown(context.Context) error                { return nil }
func (fakeLogExporter) ForceFlush(context.Context) error              { return nil }

// registerFakeGRPC registers fake factories and returns a cleanup.
func registerFakeGRPC(t *testing.T, spanErr error) {
	t.Helper()
	RegisterGRPC(
		func(context.Context, SignalConfig, TLSMode) (sdktrace.SpanExporter, error) {
			return fakeSpanExporter{exportErr: spanErr}, nil
		},
		func(context.Context, SignalConfig, TLSMode, Temporality) (sdkmetric.Exporter, error) {
			return fakeMetricExporter{}, nil
		},
		func(context.Context, SignalConfig, TLSMode) (sdklog.Exporter, error) {
			return fakeLogExporter{}, nil
		},
	)
	t.Cleanup(func() { RegisterGRPC(nil, nil, nil) })
}

func TestGRPCRegisteredPath(t *testing.T) {
	registerFakeGRPC(t, nil)
	ctx := context.Background()
	sc := SignalConfig{Enabled: true, Transport: TransportGRPC, Endpoint: "c:4317"}
	if _, err := newSpanExporter(ctx, sc, TLSModeTLS); err != nil {
		t.Errorf("span grpc: %v", err)
	}
	if _, err := newMetricExporter(ctx, sc, TemporalityDelta, TLSModeTLS); err != nil {
		t.Errorf("metric grpc: %v", err)
	}
	if _, err := newLogExporter(ctx, sc, TLSModeTLS); err != nil {
		t.Errorf("log grpc: %v", err)
	}
}

func TestExporterGRPCNotLinked(t *testing.T) {
	RegisterGRPC(nil, nil, nil) // ensure unregistered
	ctx := context.Background()
	sc := SignalConfig{Enabled: true, Transport: TransportGRPC, Endpoint: "c:4317"}
	if _, err := newSpanExporter(ctx, sc, TLSModeTLS); !errors.Is(err, ErrGRPCNotLinked) {
		t.Errorf("span: %v", err)
	}
	if _, err := newMetricExporter(ctx, sc, TemporalityCumulative, TLSModeTLS); !errors.Is(err, ErrGRPCNotLinked) {
		t.Errorf("metric: %v", err)
	}
	if _, err := newLogExporter(ctx, sc, TLSModeTLS); !errors.Is(err, ErrGRPCNotLinked) {
		t.Errorf("log: %v", err)
	}
}

func TestExporterInvalidTransport(t *testing.T) {
	ctx := context.Background()
	sc := SignalConfig{Transport: Transport(99)}
	if _, err := newSpanExporter(ctx, sc, TLSModeTLS); !errors.Is(err, ErrInvalidProtocol) {
		t.Errorf("span: %v", err)
	}
	if _, err := newMetricExporter(ctx, sc, TemporalityCumulative, TLSModeTLS); !errors.Is(err, ErrInvalidProtocol) {
		t.Errorf("metric: %v", err)
	}
	if _, err := newLogExporter(ctx, sc, TLSModeTLS); !errors.Is(err, ErrInvalidProtocol) {
		t.Errorf("log: %v", err)
	}
}

func TestExporterHTTPSkipVerifyAndDelta(t *testing.T) {
	ctx := context.Background()
	// SkipVerify exercises tlsConfigForSkipVerify across all three signals.
	sc := SignalConfig{Enabled: true, Transport: TransportHTTP, Endpoint: "c:4318", Headers: map[string]string{"authorization": "k"}}
	if _, err := newSpanExporter(ctx, sc, TLSModeSkipVerify); err != nil {
		t.Errorf("span skipverify: %v", err)
	}
	if _, err := newMetricExporter(ctx, sc, TemporalityDelta, TLSModeSkipVerify); err != nil {
		t.Errorf("metric skipverify+delta: %v", err)
	}
	if _, err := newLogExporter(ctx, sc, TLSModeSkipVerify); err != nil {
		t.Errorf("log skipverify: %v", err)
	}
}

func TestExporterHTTPPlaintext(t *testing.T) {
	ctx := context.Background()
	sc := SignalConfig{Enabled: true, Transport: TransportHTTP, Endpoint: "c:4318"}
	if _, err := newSpanExporter(ctx, sc, TLSModePlaintext); err != nil {
		t.Errorf("span plaintext: %v", err)
	}
	if _, err := newMetricExporter(ctx, sc, TemporalityCumulative, TLSModePlaintext); err != nil {
		t.Errorf("metric plaintext: %v", err)
	}
	if _, err := newLogExporter(ctx, sc, TLSModePlaintext); err != nil {
		t.Errorf("log plaintext: %v", err)
	}
}

func TestExporterHTTPEndpointError(t *testing.T) {
	ctx := context.Background()
	sc := SignalConfig{Enabled: true, Transport: TransportHTTP} // no endpoint
	if _, err := newMetricExporter(ctx, sc, TemporalityCumulative, TLSModeTLS); !errors.Is(err, ErrMissingEndpoint) {
		t.Errorf("metric: %v", err)
	}
	if _, err := newLogExporter(ctx, sc, TLSModeTLS); !errors.Is(err, ErrMissingEndpoint) {
		t.Errorf("log: %v", err)
	}
	if _, err := newSpanExporter(ctx, sc, TLSModeTLS); !errors.Is(err, ErrMissingEndpoint) {
		t.Errorf("span: %v", err)
	}
}

func TestDeltaTemporalitySelector(t *testing.T) {
	if got := deltaTemporalitySelector(sdkmetric.InstrumentKindCounter); got != metricdata.DeltaTemporality {
		t.Errorf("deltaTemporalitySelector = %v, want delta", got)
	}
}

func TestRegisterGRPCSetsFactories(t *testing.T) {
	registerFakeGRPC(t, nil)
	if grpcSpanFactory == nil || grpcMetricFactory == nil || grpcLogFactory == nil {
		t.Error("RegisterGRPC did not set factories")
	}
}

func TestInitSelfTestFailureViaGRPC(t *testing.T) {
	// A fake gRPC span exporter that errors on export makes ForceFlush (and
	// thus the self-test) fail, exercising Init's self-test failure path and
	// ForceFlush's error branch.
	registerFakeGRPC(t, errors.New("boom"))
	_, err := Init(context.Background(),
		WithEnvOverrides(false),
		WithConfig(Config{
			ServiceName: "s",
			Traces:      SignalConfig{Enabled: true, Transport: TransportGRPC, Endpoint: "c:4317"},
		}),
		WithSelfTest(),
	)
	if err == nil || !errorContains(err, "self-test failed") {
		t.Fatalf("err = %v, want self-test failure", err)
	}
}

func TestBuildTracerProviderBatchTimeout(t *testing.T) {
	registerFakeGRPC(t, nil)
	c := defaultConfig()
	c.Traces = SignalConfig{Enabled: true, Transport: TransportGRPC, Endpoint: "c:4317", BatchTimeout: 250 * time.Millisecond}
	tp, shut, err := buildTracerProvider(context.Background(), c, nil)
	if err != nil {
		t.Fatalf("buildTracerProvider: %v", err)
	}
	if tp == nil || shut == nil {
		t.Error("nil tp/shutdown")
	}
	_ = shut(context.Background())
}

func errorContains(err error, sub string) bool {
	return err != nil && contains(err.Error(), sub)
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
