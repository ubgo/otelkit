package otelkit

import (
	"errors"
	"testing"
	"time"
)

func TestEnvDisabled(t *testing.T) {
	tests := []struct {
		val  string
		want bool
	}{
		{"true", true},
		{"TRUE", true},
		{"  true  ", true},
		{"false", false},
		{"", false},
		{"1", false},
	}
	for _, tt := range tests {
		t.Setenv("OTEL_SDK_DISABLED", tt.val)
		if got := envDisabled(); got != tt.want {
			t.Errorf("envDisabled(%q) = %v, want %v", tt.val, got, tt.want)
		}
	}
}

func TestParseProtocol(t *testing.T) {
	tests := []struct {
		in     string
		want   Transport
		wantOK bool
	}{
		{"grpc", TransportGRPC, true},
		{"http/protobuf", TransportHTTP, true},
		{"http/json", TransportHTTP, true},
		{"HTTP/PROTOBUF", TransportHTTP, true},
		{"bogus", 0, false},
		{"", 0, false},
	}
	for _, tt := range tests {
		got, ok := parseProtocol(tt.in)
		if ok != tt.wantOK || (ok && got != tt.want) {
			t.Errorf("parseProtocol(%q) = (%v,%v), want (%v,%v)", tt.in, got, ok, tt.want, tt.wantOK)
		}
	}
}

func TestParseSampler(t *testing.T) {
	tests := []struct {
		in     string
		want   Sampler
		wantOK bool
	}{
		{"always_on", SamplerAlwaysOn, true},
		{"always_off", SamplerAlwaysOff, true},
		{"traceidratio", SamplerTraceIDRatio, true},
		{"parentbased_always_on", SamplerParentBasedAlwaysOn, true},
		{"parentbased_traceidratio", SamplerParentBasedTraceIDRatio, true},
		{"nope", 0, false},
	}
	for _, tt := range tests {
		got, ok := parseSampler(tt.in)
		if ok != tt.wantOK || (ok && got != tt.want) {
			t.Errorf("parseSampler(%q) = (%v,%v), want (%v,%v)", tt.in, got, ok, tt.want, tt.wantOK)
		}
	}
}

func TestParseHeaders(t *testing.T) {
	got := parseHeaders(" authorization = key123 , x-dataset=metrics ,, bad ,=novalue, k= ")
	want := map[string]string{"authorization": "key123", "x-dataset": "metrics", "k": ""}
	if len(got) != len(want) {
		t.Fatalf("parseHeaders len = %d (%v), want %d", len(got), got, len(want))
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("parseHeaders[%q] = %q, want %q", k, got[k], v)
		}
	}
}

func TestMergeHeaders(t *testing.T) {
	base := map[string]string{"a": "1", "b": "2"}
	add := map[string]string{"b": "override", "c": "3"}
	got := mergeHeaders(base, add)
	want := map[string]string{"a": "1", "b": "override", "c": "3"}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("mergeHeaders[%q] = %q, want %q", k, got[k], v)
		}
	}
	// base must be untouched
	if base["b"] != "2" {
		t.Error("mergeHeaders mutated base")
	}
}

func TestApplyEnv(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "billing")
	t.Setenv("OTEL_TRACES_SAMPLER", "traceidratio")
	t.Setenv("OTEL_TRACES_SAMPLER_ARG", "0.25")
	t.Setenv("OTEL_METRIC_EXPORT_INTERVAL", "15000")
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_TEMPORALITY_PREFERENCE", "delta")
	t.Setenv("OTEL_EXPORTER_OTLP_INSECURE", "true")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "collector:4318")
	t.Setenv("OTEL_EXPORTER_OTLP_HEADERS", "authorization=abc")

	c := defaultConfig()
	if err := applyEnv(&c); err != nil {
		t.Fatalf("applyEnv: %v", err)
	}
	if c.ServiceName != "billing" {
		t.Errorf("ServiceName = %q", c.ServiceName)
	}
	if c.Sampler != SamplerTraceIDRatio || c.SamplerRatio != 0.25 {
		t.Errorf("sampler = %v ratio %v", c.Sampler, c.SamplerRatio)
	}
	if c.MetricInterval != 15*time.Second {
		t.Errorf("MetricInterval = %v", c.MetricInterval)
	}
	if c.MetricTemporality != TemporalityDelta {
		t.Errorf("temporality = %v", c.MetricTemporality)
	}
	if c.TLS != TLSModePlaintext {
		t.Errorf("TLS = %v", c.TLS)
	}
	if !c.Traces.Enabled || c.Traces.Transport != TransportHTTP || c.Traces.Endpoint != "collector:4318" {
		t.Errorf("traces = %+v", c.Traces)
	}
	if c.Traces.Headers["authorization"] != "abc" {
		t.Errorf("traces headers = %v", c.Traces.Headers)
	}
}

func TestApplyEnvPerSignalOverride(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "generic:4318")
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "http/protobuf")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "https://traces.example/v1/traces")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_PROTOCOL", "grpc")
	t.Setenv("OTEL_EXPORTER_OTLP_TRACES_HEADERS", "x=1")

	c := defaultConfig()
	if err := applyEnv(&c); err != nil {
		t.Fatalf("applyEnv: %v", err)
	}
	// Per-signal endpoint is verbatim.
	if !c.Traces.EndpointIsURL || c.Traces.Endpoint != "https://traces.example/v1/traces" {
		t.Errorf("traces endpoint = %q isURL=%v", c.Traces.Endpoint, c.Traces.EndpointIsURL)
	}
	if c.Traces.Transport != TransportGRPC {
		t.Errorf("traces transport = %v, want grpc", c.Traces.Transport)
	}
	if c.Traces.Headers["x"] != "1" {
		t.Errorf("traces headers = %v", c.Traces.Headers)
	}
	// Metrics inherits the generic endpoint (not verbatim).
	if c.Metrics.EndpointIsURL || c.Metrics.Endpoint != "generic:4318" {
		t.Errorf("metrics endpoint = %q isURL=%v", c.Metrics.Endpoint, c.Metrics.EndpointIsURL)
	}
}

func TestApplyEnvInvalidProtocol(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_PROTOCOL", "carrier-pigeon")
	c := defaultConfig()
	if err := applyEnv(&c); !errors.Is(err, ErrInvalidProtocol) {
		t.Fatalf("err = %v, want ErrInvalidProtocol", err)
	}
}

func TestApplyEnvInvalidPerSignalProtocol(t *testing.T) {
	// One case per signal so each applySignalEnv error-return is exercised.
	for _, key := range []string{"TRACES", "METRICS", "LOGS"} {
		t.Run(key, func(t *testing.T) {
			t.Setenv("OTEL_EXPORTER_OTLP_"+key+"_PROTOCOL", "smoke-signal")
			c := defaultConfig()
			if err := applyEnv(&c); !errors.Is(err, ErrInvalidProtocol) {
				t.Fatalf("err = %v, want ErrInvalidProtocol", err)
			}
		})
	}
}

func TestApplyEnvEmptyIsNoop(t *testing.T) {
	// With no OTEL_* set, applyEnv must not enable anything or error.
	for _, k := range []string{
		"OTEL_SERVICE_NAME", "OTEL_TRACES_SAMPLER", "OTEL_TRACES_SAMPLER_ARG",
		"OTEL_METRIC_EXPORT_INTERVAL", "OTEL_EXPORTER_OTLP_METRICS_TEMPORALITY_PREFERENCE",
		"OTEL_EXPORTER_OTLP_INSECURE", "OTEL_EXPORTER_OTLP_PROTOCOL",
		"OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_EXPORTER_OTLP_HEADERS",
	} {
		t.Setenv(k, "")
	}
	c := defaultConfig()
	if err := applyEnv(&c); err != nil {
		t.Fatalf("applyEnv: %v", err)
	}
	if c.Traces.Enabled || c.Metrics.Enabled || c.Logs.Enabled {
		t.Error("empty env enabled a signal")
	}
}

func TestApplyEnvInvalidNumbersIgnored(t *testing.T) {
	t.Setenv("OTEL_TRACES_SAMPLER_ARG", "notafloat")
	t.Setenv("OTEL_METRIC_EXPORT_INTERVAL", "0")
	t.Setenv("OTEL_EXPORTER_OTLP_METRICS_TEMPORALITY_PREFERENCE", "cumulative")
	c := defaultConfig()
	if err := applyEnv(&c); err != nil {
		t.Fatalf("applyEnv: %v", err)
	}
	if c.SamplerRatio != 1.0 {
		t.Errorf("SamplerRatio = %v, want unchanged 1.0", c.SamplerRatio)
	}
	if c.MetricInterval != defaultMetricInterval {
		t.Errorf("MetricInterval = %v, want unchanged", c.MetricInterval)
	}
	if c.MetricTemporality != TemporalityCumulative {
		t.Errorf("temporality = %v", c.MetricTemporality)
	}
}

func TestApplyEnvNonNumericInterval(t *testing.T) {
	t.Setenv("OTEL_METRIC_EXPORT_INTERVAL", "abc")
	c := defaultConfig()
	if err := applyEnv(&c); err != nil {
		t.Fatalf("applyEnv: %v", err)
	}
	if c.MetricInterval != defaultMetricInterval {
		t.Errorf("MetricInterval = %v, want unchanged on parse error", c.MetricInterval)
	}
}

func TestApplyEnvUnknownSamplerKeepsDefault(t *testing.T) {
	t.Setenv("OTEL_TRACES_SAMPLER", "mystery")
	c := defaultConfig()
	if err := applyEnv(&c); err != nil {
		t.Fatalf("applyEnv: %v", err)
	}
	if c.Sampler != SamplerParentBasedAlwaysOn {
		t.Errorf("Sampler = %v, want default", c.Sampler)
	}
}
