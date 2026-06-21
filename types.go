// types.go — the small enums that describe a telemetry pipeline: Signal,
// Transport, Sampler, Temporality, TLSMode.
//
// Each enum carries the OTEL-spec mappings the rest of the package relies on:
// Transport knows its default OTLP port (4317/4318) and protocol value; Signal
// knows its "/v1/<signal>" HTTP path suffix; Sampler stringifies to the
// OTEL_TRACES_SAMPLER values. Pure value types, no behaviour beyond String()
// and these lookups. Used by config.go (endpoint resolution) and providers.go
// (building the SDK objects).

package otelkit

import "fmt"

// Signal identifies one of the three OpenTelemetry signals.
type Signal int

const (
	// SignalTraces is the distributed-tracing signal.
	SignalTraces Signal = iota
	// SignalMetrics is the metrics signal.
	SignalMetrics
	// SignalLogs is the logs signal.
	SignalLogs
)

// String returns the lowercase signal name ("traces"/"metrics"/"logs"), or
// "signal(N)" for an unknown value.
func (s Signal) String() string {
	switch s {
	case SignalTraces:
		return "traces"
	case SignalMetrics:
		return "metrics"
	case SignalLogs:
		return "logs"
	default:
		return fmt.Sprintf("signal(%d)", int(s))
	}
}

// pathSuffix is the OTLP/HTTP path appended to a generic endpoint for this
// signal (e.g. "/v1/traces"). Per the OTLP spec the suffix is appended only to
// the signal-agnostic endpoint; per-signal endpoints are used verbatim.
func (s Signal) pathSuffix() string {
	switch s {
	case SignalTraces:
		return "/v1/traces"
	case SignalMetrics:
		return "/v1/metrics"
	case SignalLogs:
		return "/v1/logs"
	default:
		return ""
	}
}

// Transport selects how a signal's OTLP exporter talks to the backend.
type Transport int

const (
	// TransportStdout writes telemetry to stdout — for local dev and the
	// dry-run path. No network, no auth.
	TransportStdout Transport = iota
	// TransportHTTP is OTLP over HTTP (default OTLP port 4318).
	TransportHTTP
	// TransportGRPC is OTLP over gRPC (default OTLP port 4317). The gRPC
	// exporters ship in the contrib/otelkit-grpc module to keep core's
	// dependency graph free of google.golang.org/grpc.
	TransportGRPC
)

// String returns the canonical transport name.
func (t Transport) String() string {
	switch t {
	case TransportStdout:
		return "stdout"
	case TransportHTTP:
		return "http"
	case TransportGRPC:
		return "grpc"
	default:
		return fmt.Sprintf("transport(%d)", int(t))
	}
}

// defaultPort returns the conventional OTLP port for the transport, or 0 when
// the transport is portless (stdout).
func (t Transport) defaultPort() int {
	switch t {
	case TransportHTTP:
		return 4318
	case TransportGRPC:
		return 4317
	default:
		return 0
	}
}

// protocolValue maps the transport to the spec OTEL_EXPORTER_OTLP_PROTOCOL
// value. stdout has no OTLP protocol and returns "".
func (t Transport) protocolValue() string {
	switch t {
	case TransportHTTP:
		return "http/protobuf"
	case TransportGRPC:
		return "grpc"
	default:
		return ""
	}
}

// Sampler selects the head sampling strategy. Tail sampling is a Collector
// concern and is intentionally out of scope.
type Sampler int

const (
	// SamplerParentBasedAlwaysOn samples every root span and respects the
	// parent's decision for child spans. This is the OTEL default and keeps
	// traces unbroken across services.
	SamplerParentBasedAlwaysOn Sampler = iota
	// SamplerAlwaysOn samples every span.
	SamplerAlwaysOn
	// SamplerAlwaysOff samples no spans.
	SamplerAlwaysOff
	// SamplerTraceIDRatio samples a fraction of root spans (see Config.SamplerRatio).
	SamplerTraceIDRatio
	// SamplerParentBasedTraceIDRatio is ratio sampling for roots, parent
	// decision for children.
	SamplerParentBasedTraceIDRatio
)

// String returns the spec OTEL_TRACES_SAMPLER value for the sampler.
func (s Sampler) String() string {
	switch s {
	case SamplerParentBasedAlwaysOn:
		return "parentbased_always_on"
	case SamplerAlwaysOn:
		return "always_on"
	case SamplerAlwaysOff:
		return "always_off"
	case SamplerTraceIDRatio:
		return "traceidratio"
	case SamplerParentBasedTraceIDRatio:
		return "parentbased_traceidratio"
	default:
		return fmt.Sprintf("sampler(%d)", int(s))
	}
}

// Temporality selects the metric aggregation temporality. Some backends
// (Datadog direct intake) reject cumulative and require delta; presets set
// this correctly per vendor.
type Temporality int

const (
	// TemporalityCumulative is the OTEL default for all instrument kinds.
	TemporalityCumulative Temporality = iota
	// TemporalityDelta reports deltas between collections. Required by
	// Datadog direct OTLP intake; preferred by New Relic and Uptrace.
	TemporalityDelta
)

// String returns the canonical temporality name.
func (t Temporality) String() string {
	switch t {
	case TemporalityCumulative:
		return "cumulative"
	case TemporalityDelta:
		return "delta"
	default:
		return fmt.Sprintf("temporality(%d)", int(t))
	}
}

// TLSMode is the transport security mode for OTLP exporters. The three modes
// are mutually exclusive, so a contradictory combination (e.g. plaintext with
// a client certificate) is rejected at construction rather than failing
// silently at export time.
type TLSMode int

const (
	// TLSModeTLS uses TLS with full server-certificate verification (the
	// default for non-stdout transports).
	TLSModeTLS TLSMode = iota
	// TLSModePlaintext disables TLS entirely (cleartext). Use only for a
	// local collector on a trusted network.
	TLSModePlaintext
	// TLSModeSkipVerify uses TLS but skips server-certificate verification.
	// Insecure; for self-signed collectors in dev only.
	TLSModeSkipVerify
)

// String returns the canonical TLS-mode name.
func (m TLSMode) String() string {
	switch m {
	case TLSModeTLS:
		return "tls"
	case TLSModePlaintext:
		return "plaintext"
	case TLSModeSkipVerify:
		return "skip-verify"
	default:
		return fmt.Sprintf("tlsmode(%d)", int(m))
	}
}
