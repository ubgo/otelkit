package otelkit

import "testing"

func TestSignalString(t *testing.T) {
	tests := []struct {
		in   Signal
		want string
	}{
		{SignalTraces, "traces"},
		{SignalMetrics, "metrics"},
		{SignalLogs, "logs"},
		{Signal(99), "signal(99)"},
	}
	for _, tt := range tests {
		if got := tt.in.String(); got != tt.want {
			t.Errorf("Signal(%d).String() = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestSignalPathSuffix(t *testing.T) {
	tests := []struct {
		in   Signal
		want string
	}{
		{SignalTraces, "/v1/traces"},
		{SignalMetrics, "/v1/metrics"},
		{SignalLogs, "/v1/logs"},
		{Signal(99), ""},
	}
	for _, tt := range tests {
		if got := tt.in.pathSuffix(); got != tt.want {
			t.Errorf("Signal(%d).pathSuffix() = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestTransportString(t *testing.T) {
	tests := []struct {
		in   Transport
		want string
	}{
		{TransportStdout, "stdout"},
		{TransportHTTP, "http"},
		{TransportGRPC, "grpc"},
		{Transport(99), "transport(99)"},
	}
	for _, tt := range tests {
		if got := tt.in.String(); got != tt.want {
			t.Errorf("Transport(%d).String() = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestTransportDefaultPort(t *testing.T) {
	tests := []struct {
		in   Transport
		want int
	}{
		{TransportHTTP, 4318},
		{TransportGRPC, 4317},
		{TransportStdout, 0},
		{Transport(99), 0},
	}
	for _, tt := range tests {
		if got := tt.in.defaultPort(); got != tt.want {
			t.Errorf("Transport(%d).defaultPort() = %d, want %d", tt.in, got, tt.want)
		}
	}
}

func TestTransportProtocolValue(t *testing.T) {
	tests := []struct {
		in   Transport
		want string
	}{
		{TransportHTTP, "http/protobuf"},
		{TransportGRPC, "grpc"},
		{TransportStdout, ""},
		{Transport(99), ""},
	}
	for _, tt := range tests {
		if got := tt.in.protocolValue(); got != tt.want {
			t.Errorf("Transport(%d).protocolValue() = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestSamplerString(t *testing.T) {
	tests := []struct {
		in   Sampler
		want string
	}{
		{SamplerParentBasedAlwaysOn, "parentbased_always_on"},
		{SamplerAlwaysOn, "always_on"},
		{SamplerAlwaysOff, "always_off"},
		{SamplerTraceIDRatio, "traceidratio"},
		{SamplerParentBasedTraceIDRatio, "parentbased_traceidratio"},
		{Sampler(99), "sampler(99)"},
	}
	for _, tt := range tests {
		if got := tt.in.String(); got != tt.want {
			t.Errorf("Sampler(%d).String() = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestTemporalityString(t *testing.T) {
	tests := []struct {
		in   Temporality
		want string
	}{
		{TemporalityCumulative, "cumulative"},
		{TemporalityDelta, "delta"},
		{Temporality(99), "temporality(99)"},
	}
	for _, tt := range tests {
		if got := tt.in.String(); got != tt.want {
			t.Errorf("Temporality(%d).String() = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestTLSModeString(t *testing.T) {
	tests := []struct {
		in   TLSMode
		want string
	}{
		{TLSModeTLS, "tls"},
		{TLSModePlaintext, "plaintext"},
		{TLSModeSkipVerify, "skip-verify"},
		{TLSMode(99), "tlsmode(99)"},
	}
	for _, tt := range tests {
		if got := tt.in.String(); got != tt.want {
			t.Errorf("TLSMode(%d).String() = %q, want %q", tt.in, got, tt.want)
		}
	}
}
