package otelkitgrpc

import (
	"context"
	"errors"
	"testing"

	"github.com/ubgo/otelkit"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// Importing this package runs init → Register, so otelkit can build gRPC
// exporters end to end.

func TestInitWithGRPC(t *testing.T) {
	tel, err := otelkit.Init(context.Background(),
		otelkit.WithEnvOverrides(false),
		otelkit.WithService("svc", "1"),
		otelkit.WithPreset(otelkit.PresetCollector("localhost:4317", otelkit.TransportGRPC)),
	)
	if err != nil {
		t.Fatalf("Init with grpc: %v", err)
	}
	_ = tel.Shutdown(context.Background())
}

func TestRegisterEnablesGRPC(t *testing.T) {
	Register()
	ctx := context.Background()
	sc := otelkit.SignalConfig{Enabled: true, Transport: otelkit.TransportGRPC, Endpoint: "c:4317", Headers: map[string]string{"authorization": "k"}}

	if _, err := newSpanExporter(ctx, sc, otelkit.TLSModeTLS); err != nil {
		t.Errorf("span tls: %v", err)
	}
	if _, err := newSpanExporter(ctx, sc, otelkit.TLSModePlaintext); err != nil {
		t.Errorf("span plaintext: %v", err)
	}
	if _, err := newSpanExporter(ctx, sc, otelkit.TLSModeSkipVerify); err != nil {
		t.Errorf("span skipverify: %v", err)
	}
	if _, err := newMetricExporter(ctx, sc, otelkit.TLSModePlaintext, otelkit.TemporalityDelta); err != nil {
		t.Errorf("metric delta plaintext: %v", err)
	}
	if _, err := newMetricExporter(ctx, sc, otelkit.TLSModeSkipVerify, otelkit.TemporalityCumulative); err != nil {
		t.Errorf("metric skipverify: %v", err)
	}
	if _, err := newMetricExporter(ctx, sc, otelkit.TLSModeTLS, otelkit.TemporalityCumulative); err != nil {
		t.Errorf("metric tls: %v", err)
	}
	if _, err := newLogExporter(ctx, sc, otelkit.TLSModePlaintext); err != nil {
		t.Errorf("log plaintext: %v", err)
	}
	if _, err := newLogExporter(ctx, sc, otelkit.TLSModeSkipVerify); err != nil {
		t.Errorf("log skipverify: %v", err)
	}
}

func TestGRPCExportersMissingEndpoint(t *testing.T) {
	ctx := context.Background()
	sc := otelkit.SignalConfig{Enabled: true, Transport: otelkit.TransportGRPC} // no endpoint
	if _, err := newSpanExporter(ctx, sc, otelkit.TLSModeTLS); !errors.Is(err, otelkit.ErrMissingEndpoint) {
		t.Errorf("span: %v", err)
	}
	if _, err := newMetricExporter(ctx, sc, otelkit.TLSModeTLS, otelkit.TemporalityCumulative); !errors.Is(err, otelkit.ErrMissingEndpoint) {
		t.Errorf("metric: %v", err)
	}
	if _, err := newLogExporter(ctx, sc, otelkit.TLSModeTLS); !errors.Is(err, otelkit.ErrMissingEndpoint) {
		t.Errorf("log: %v", err)
	}
}

func TestSkipVerifyCreds(t *testing.T) {
	if skipVerifyCreds() == nil {
		t.Error("nil creds")
	}
}

func TestDeltaSelector(t *testing.T) {
	if got := deltaSelector(sdkmetric.InstrumentKindCounter); got != metricdata.DeltaTemporality {
		t.Errorf("deltaSelector = %v, want delta", got)
	}
}
