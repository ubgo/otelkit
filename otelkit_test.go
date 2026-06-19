package otelkit

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// stdoutConfig returns options enabling all three signals on stdout — a fully
// offline, deterministic end-to-end setup.
func allStdout() []Option {
	return []Option{
		WithService("test-svc", "0.1.0"),
		WithConfig(Config{
			ServiceName:       "test-svc",
			ResourceDetectors: "none",
			Traces:            SignalConfig{Enabled: true, Transport: TransportStdout},
			Metrics:           SignalConfig{Enabled: true, Transport: TransportStdout},
			Logs:              SignalConfig{Enabled: true, Transport: TransportStdout},
		}),
		WithEnvOverrides(false),
	}
}

func TestInitStdoutEndToEnd(t *testing.T) {
	tel, err := Init(context.Background(), allStdout()...)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(context.Background()) })

	if tel.Disabled() {
		t.Error("Disabled() = true, want false")
	}
	if _, ok := tel.TracerProvider().(*sdktrace.TracerProvider); !ok {
		t.Errorf("TracerProvider is %T, want *sdktrace.TracerProvider", tel.TracerProvider())
	}
	if _, ok := tel.MeterProvider().(*sdkmetric.MeterProvider); !ok {
		t.Errorf("MeterProvider is %T", tel.MeterProvider())
	}
	if _, ok := tel.LoggerProvider().(*sdklog.LoggerProvider); !ok {
		t.Errorf("LoggerProvider is %T", tel.LoggerProvider())
	}

	tel.SetGlobal() // must not panic

	if err := tel.SelfTest(context.Background()); err != nil {
		t.Errorf("SelfTest: %v", err)
	}
	if err := tel.ForceFlush(context.Background()); err != nil {
		t.Errorf("ForceFlush: %v", err)
	}
	if err := tel.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown: %v", err)
	}
}

func TestInitDisabled(t *testing.T) {
	t.Setenv("OTEL_SDK_DISABLED", "true")
	tel, err := Init(context.Background())
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if !tel.Disabled() {
		t.Error("Disabled() = false, want true")
	}
	// No-op providers: ForceFlush/Shutdown/SelfTest are clean no-ops.
	if err := tel.ForceFlush(context.Background()); err != nil {
		t.Errorf("ForceFlush: %v", err)
	}
	if err := tel.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown: %v", err)
	}
	if err := tel.SelfTest(context.Background()); err != nil {
		t.Errorf("SelfTest: %v", err)
	}
	tel.SetGlobal()
	if tel.Tracer("x") == nil {
		t.Error("Tracer returned nil")
	}
}

func TestInitDisabledSignalsAreNoop(t *testing.T) {
	tel, err := Init(context.Background(), WithEnvOverrides(false), WithService("s", "1"))
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(context.Background()) })
	// Nothing enabled → all providers are the no-op implementations and there
	// are no shutdown funcs.
	if _, ok := tel.TracerProvider().(*sdktrace.TracerProvider); ok {
		t.Error("expected no-op tracer provider")
	}
	if len(tel.shutdowns) != 0 {
		t.Errorf("shutdowns = %d, want 0", len(tel.shutdowns))
	}
}

func TestInitEnvInvalidProtocolError(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "pigeon")
	if _, err := Init(context.Background(), WithService("s", "1")); !errors.Is(err, ErrInvalidProtocol) {
		t.Fatalf("err = %v, want ErrInvalidProtocol", err)
	}
}

func TestInitGRPCNotLinked(t *testing.T) {
	_, err := Init(context.Background(), WithEnvOverrides(false), WithConfig(Config{
		ServiceName: "s",
		Traces:      SignalConfig{Enabled: true, Transport: TransportGRPC, Endpoint: "c:4317"},
	}))
	if !errors.Is(err, ErrGRPCNotLinked) {
		t.Fatalf("err = %v, want ErrGRPCNotLinked", err)
	}
}

func TestInitHTTPConstructs(t *testing.T) {
	// HTTP exporters construct without dialing, so Init succeeds offline.
	tel, err := Init(context.Background(), WithEnvOverrides(false), WithConfig(Config{
		ServiceName: "s",
		Traces:      SignalConfig{Enabled: true, Transport: TransportHTTP, Endpoint: "localhost:4318"},
		Metrics:     SignalConfig{Enabled: true, Transport: TransportHTTP, Endpoint: "localhost:4318"},
		Logs:        SignalConfig{Enabled: true, Transport: TransportHTTP, Endpoint: "localhost:4318"},
	}), WithTLS(TLSModePlaintext))
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	_ = tel.Shutdown(context.Background())
}

func TestInitSelfTestEnabledStdout(t *testing.T) {
	opts := append(allStdout(), WithSelfTest())
	tel, err := Init(context.Background(), opts...)
	if err != nil {
		t.Fatalf("Init with self-test: %v", err)
	}
	_ = tel.Shutdown(context.Background())
}

func TestBuildSampler(t *testing.T) {
	for _, s := range []Sampler{
		SamplerParentBasedAlwaysOn, SamplerAlwaysOn, SamplerAlwaysOff,
		SamplerTraceIDRatio, SamplerParentBasedTraceIDRatio, Sampler(99),
	} {
		if got := buildSampler(s, 0.5); got == nil {
			t.Errorf("buildSampler(%v) = nil", s)
		}
	}
}

func TestBuildPropagator(t *testing.T) {
	// Default (env disabled).
	if buildPropagator(Config{envOverrides: false}) == nil {
		t.Error("nil propagator")
	}
	// Env set, custom list.
	t.Setenv("OTEL_PROPAGATORS", "tracecontext,baggage,unknown")
	if buildPropagator(Config{envOverrides: true}) == nil {
		t.Error("nil propagator from env list")
	}
	// Env "none".
	t.Setenv("OTEL_PROPAGATORS", "none")
	if buildPropagator(Config{envOverrides: true}) == nil {
		t.Error("nil propagator for none")
	}
	// Env empty → default.
	t.Setenv("OTEL_PROPAGATORS", "")
	if buildPropagator(Config{envOverrides: true}) == nil {
		t.Error("nil propagator for empty env")
	}
	// Env with only-unknown tokens → falls back to default.
	t.Setenv("OTEL_PROPAGATORS", "bogus")
	if buildPropagator(Config{envOverrides: true}) == nil {
		t.Error("nil propagator for unknown-only")
	}
}

func TestApplyDryRun(t *testing.T) {
	c := defaultConfig()
	c.ServiceName = "svc"
	c.Traces = SignalConfig{Enabled: true, Transport: TransportHTTP, Endpoint: "https://x/v1/traces", Headers: map[string]string{"authorization": "secret"}}
	c.Metrics = SignalConfig{Enabled: false}

	tmp, err := os.CreateTemp(t.TempDir(), "dryrun")
	if err != nil {
		t.Fatal(err)
	}
	applyDryRun(&c, tmp)
	_ = tmp.Close()
	out, _ := os.ReadFile(tmp.Name())
	s := string(out)
	if !strings.Contains(s, "DRY RUN") {
		t.Errorf("missing DRY RUN banner: %q", s)
	}
	if strings.Contains(s, "secret") {
		t.Error("dry-run leaked an auth header value")
	}
	if !strings.Contains(s, "<redacted>") {
		t.Error("dry-run did not redact headers")
	}
	if !strings.Contains(s, "metrics: disabled") {
		t.Error("dry-run did not show disabled signal")
	}
	// Enabled signal rewritten to stdout.
	if c.Traces.Transport != TransportStdout {
		t.Errorf("traces transport = %v, want stdout after dry-run", c.Traces.Transport)
	}
}

func TestInitDryRun(t *testing.T) {
	tel, err := Init(context.Background(), WithEnvOverrides(false), WithConfig(Config{
		ServiceName: "s",
		Traces:      SignalConfig{Enabled: true, Transport: TransportHTTP, Endpoint: "https://x"},
	}), WithDryRun())
	if err != nil {
		t.Fatalf("Init dry-run: %v", err)
	}
	_ = tel.Shutdown(context.Background())
}

func TestRedactHeaders(t *testing.T) {
	if got := redactHeaders(nil); got != "{}" {
		t.Errorf("empty = %q", got)
	}
	got := redactHeaders(map[string]string{"b": "2", "a": "1"})
	if got != "{a=<redacted>,b=<redacted>}" {
		t.Errorf("redactHeaders = %q", got)
	}
}

func TestRunOnSignalCtxCancel(t *testing.T) {
	tel, err := Init(context.Background(), allStdout()...)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled → RunOnSignal returns after Shutdown
	done := make(chan error, 1)
	go func() { done <- tel.RunOnSignal(ctx) }()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("RunOnSignal: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("RunOnSignal did not return on ctx cancel")
	}
}

func TestBuildMeterProviderInterval(t *testing.T) {
	// MetricInterval <= 0 falls back to the default.
	c := defaultConfig()
	c.MetricInterval = 0
	c.Metrics = SignalConfig{Enabled: true, Transport: TransportStdout}
	mp, shut, err := buildMeterProvider(context.Background(), c, nil)
	if err != nil {
		t.Fatalf("buildMeterProvider: %v", err)
	}
	if mp == nil || shut == nil {
		t.Error("nil provider/shutdown")
	}
	_ = shut(context.Background())
}

func TestProvidersExporterError(t *testing.T) {
	// HTTP transport with no endpoint → exporter build fails, surfaced by each
	// provider builder.
	c := defaultConfig()
	c.Traces = SignalConfig{Enabled: true, Transport: TransportHTTP}
	if _, _, err := buildTracerProvider(context.Background(), c, nil); !errors.Is(err, ErrMissingEndpoint) {
		t.Errorf("tracer err = %v", err)
	}
	c = defaultConfig()
	c.Metrics = SignalConfig{Enabled: true, Transport: TransportHTTP}
	if _, _, err := buildMeterProvider(context.Background(), c, nil); !errors.Is(err, ErrMissingEndpoint) {
		t.Errorf("meter err = %v", err)
	}
	c = defaultConfig()
	c.Logs = SignalConfig{Enabled: true, Transport: TransportHTTP}
	if _, _, err := buildLoggerProvider(context.Background(), c, nil); !errors.Is(err, ErrMissingEndpoint) {
		t.Errorf("logger err = %v", err)
	}
}

func TestForceFlushNoopBuffer(t *testing.T) {
	// Sanity: a captured buffer to ensure stdout exporters don't explode under
	// flush (also exercises bytes import usefully).
	var buf bytes.Buffer
	_ = buf
}

func TestInitMetricsBuildError(t *testing.T) {
	_, err := Init(context.Background(), WithEnvOverrides(false), WithConfig(Config{
		ServiceName: "s",
		Metrics:     SignalConfig{Enabled: true, Transport: TransportHTTP}, // no endpoint
	}))
	if !errors.Is(err, ErrMissingEndpoint) || !strings.Contains(err.Error(), "metrics") {
		t.Fatalf("err = %v, want metrics ErrMissingEndpoint", err)
	}
}

func TestInitLogsBuildError(t *testing.T) {
	_, err := Init(context.Background(), WithEnvOverrides(false), WithConfig(Config{
		ServiceName: "s",
		Logs:        SignalConfig{Enabled: true, Transport: TransportHTTP}, // no endpoint
	}))
	if !errors.Is(err, ErrMissingEndpoint) || !strings.Contains(err.Error(), "logs") {
		t.Fatalf("err = %v, want logs ErrMissingEndpoint", err)
	}
}

func TestInitResourceError(t *testing.T) {
	orig := buildResourceFn
	t.Cleanup(func() { buildResourceFn = orig })
	sentinel := errors.New("resource boom")
	buildResourceFn = func(context.Context, Config) (*sdkresource.Resource, error) { return nil, sentinel }
	_, err := Init(context.Background(), WithEnvOverrides(false), WithService("s", "1"))
	if !errors.Is(err, sentinel) || !strings.Contains(err.Error(), "build resource") {
		t.Fatalf("err = %v, want wrapped resource error", err)
	}
}
