package otelkit

import (
	"errors"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	c := defaultConfig()
	if c.Sampler != SamplerParentBasedAlwaysOn {
		t.Errorf("default Sampler = %v, want parentbased_always_on", c.Sampler)
	}
	if c.SamplerRatio != 1.0 {
		t.Errorf("default SamplerRatio = %v, want 1.0", c.SamplerRatio)
	}
	if c.MetricTemporality != TemporalityCumulative {
		t.Errorf("default MetricTemporality = %v, want cumulative", c.MetricTemporality)
	}
	if c.MetricInterval != defaultMetricInterval {
		t.Errorf("default MetricInterval = %v, want %v", c.MetricInterval, defaultMetricInterval)
	}
	if c.TLS != TLSModeTLS {
		t.Errorf("default TLS = %v, want tls", c.TLS)
	}
	if !c.envOverrides {
		t.Error("default envOverrides = false, want true")
	}
}

func TestResolveEndpoint(t *testing.T) {
	tests := []struct {
		name    string
		sc      SignalConfig
		sig     Signal
		tls     TLSMode
		want    string
		wantErr error
	}{
		// stdout
		{"stdout has no endpoint", SignalConfig{Transport: TransportStdout}, SignalTraces, TLSModeTLS, "", nil},

		// gRPC
		{"grpc host only defaults port", SignalConfig{Transport: TransportGRPC, Endpoint: "collector"}, SignalTraces, TLSModeTLS, "collector:4317", nil},
		{"grpc host:port kept", SignalConfig{Transport: TransportGRPC, Endpoint: "collector:5555"}, SignalMetrics, TLSModeTLS, "collector:5555", nil},
		{"grpc strips scheme+path", SignalConfig{Transport: TransportGRPC, Endpoint: "https://api.uptrace.dev:4317/x"}, SignalTraces, TLSModeTLS, "api.uptrace.dev:4317", nil},
		{"grpc missing endpoint errors", SignalConfig{Transport: TransportGRPC}, SignalTraces, TLSModeTLS, "", ErrMissingEndpoint},

		// HTTP generic — append /v1/<signal>
		{"http host appends path + default port + https", SignalConfig{Transport: TransportHTTP, Endpoint: "in-otel.hyperdx.io"}, SignalTraces, TLSModeTLS, "https://in-otel.hyperdx.io:4318/v1/traces", nil},
		{"http plaintext uses http scheme", SignalConfig{Transport: TransportHTTP, Endpoint: "localhost:4318"}, SignalLogs, TLSModePlaintext, "http://localhost:4318/v1/logs", nil},
		{"http base path /otlp gets /v1/metrics appended (grafana)", SignalConfig{Transport: TransportHTTP, Endpoint: "https://otlp-gateway.grafana.net/otlp"}, SignalMetrics, TLSModeTLS, "https://otlp-gateway.grafana.net:4318/otlp/v1/metrics", nil},
		{"http no double-append when suffix present", SignalConfig{Transport: TransportHTTP, Endpoint: "https://c.example/v1/traces"}, SignalTraces, TLSModeTLS, "https://c.example:4318/v1/traces", nil},
		{"http explicit port kept", SignalConfig{Transport: TransportHTTP, Endpoint: "https://c.example:443"}, SignalTraces, TLSModeTLS, "https://c.example:443/v1/traces", nil},

		// HTTP per-signal URL — verbatim, no append
		{"http EndpointIsURL verbatim", SignalConfig{Transport: TransportHTTP, Endpoint: "https://otlp.datadoghq.com/v1/metrics", EndpointIsURL: true}, SignalMetrics, TLSModeTLS, "https://otlp.datadoghq.com/v1/metrics", nil},

		// errors
		{"http missing endpoint errors", SignalConfig{Transport: TransportHTTP}, SignalTraces, TLSModeTLS, "", ErrMissingEndpoint},
		{"unknown transport errors", SignalConfig{Transport: Transport(99), Endpoint: "x"}, SignalTraces, TLSModeTLS, "", ErrInvalidProtocol},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.sc.resolveEndpoint(tt.sig, tt.tls)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got != tt.want {
				t.Errorf("resolveEndpoint() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGRPCTarget(t *testing.T) {
	got, err := SignalConfig{Endpoint: "collector"}.GRPCTarget()
	if err != nil || got != "collector:4317" {
		t.Errorf("GRPCTarget() = (%q,%v)", got, err)
	}
	if _, err := (SignalConfig{}).GRPCTarget(); !errors.Is(err, ErrMissingEndpoint) {
		t.Errorf("empty GRPCTarget err = %v", err)
	}
}

func TestHTTPURLParseError(t *testing.T) {
	sc := SignalConfig{Transport: TransportHTTP, Endpoint: "http://[::1:bad"}
	if _, err := sc.resolveEndpoint(SignalTraces, TLSModeTLS); err == nil {
		t.Error("expected parse error for malformed URL")
	}
}

func TestHTTPURLEmptyHost(t *testing.T) {
	// A URL with a scheme but no host hits the "no host" error branch.
	sc := SignalConfig{Transport: TransportHTTP, Endpoint: "https:///onlypath"}
	if _, err := sc.resolveEndpoint(SignalTraces, TLSModeTLS); err == nil {
		t.Error("expected error for endpoint with no host")
	}
}

func TestGRPCParseError(t *testing.T) {
	sc := SignalConfig{Transport: TransportGRPC, Endpoint: "://["}
	if _, err := sc.resolveEndpoint(SignalTraces, TLSModeTLS); err == nil {
		t.Error("expected parse error for malformed grpc endpoint")
	}
}

func TestGRPCEmptyHostFallsBackToPath(t *testing.T) {
	// A scheme with an empty host (file:///sock) falls back to the path so a
	// socket-ish endpoint still yields a usable host:port string.
	sc := SignalConfig{Transport: TransportGRPC, Endpoint: "file:///sock"}
	got, err := sc.resolveEndpoint(SignalTraces, TLSModeTLS)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != "/sock:4317" {
		t.Errorf("resolveEndpoint() = %q, want %q", got, "/sock:4317")
	}
}
