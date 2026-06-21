// config.go — Config, SignalConfig, the defaults, and the endpoint/path
// resolver (resolveEndpoint / GRPCTarget).
//
// This is where otelkit "owns all port + /v1/<signal> construction" — the
// single biggest silent-failure killer. resolveEndpoint is pure string logic
// (gRPC host:port, HTTP path-append with double-append guard, per-signal URLs
// verbatim) and is exhaustively unit-tested in config_test.go. Config is built
// by options.go (programmatic), env.go (OTEL_* overlay), and presets.go, then
// consumed by providers.go.

package otelkit

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
)

// Default timing values (OTEL spec defaults).
const (
	defaultExportTimeout  = 10 * time.Second
	defaultBatchTimeout   = 5 * time.Second
	defaultMetricInterval = 60 * time.Second
	defaultWatchdogGrace  = time.Second
)

// Config is the full bootstrap configuration. Build it directly, or start from
// a Preset and override fields with the With* options.
type Config struct {
	// ServiceName populates the required service.name resource attribute.
	// When empty, otelkit falls back to "unknown_service:<binary>".
	ServiceName string
	// ServiceVersion populates service.version (recommended).
	ServiceVersion string
	// Environment populates deployment.environment.name (optional).
	Environment string

	// Per-signal exporter configuration.
	Traces  SignalConfig
	Metrics SignalConfig
	Logs    SignalConfig

	// Sampler selects head sampling. Zero value = parentbased_always_on.
	Sampler Sampler
	// SamplerRatio is the [0,1] probability for ratio samplers. Default 1.0.
	SamplerRatio float64

	// MetricTemporality selects cumulative (default) or delta.
	MetricTemporality Temporality
	// MetricInterval is the periodic reader interval. 0 → 60s.
	MetricInterval time.Duration

	// TLS is the transport security mode for network exporters.
	TLS TLSMode
	// ResourceDetectors is a token list ("env,host,os,process,container",
	// "all", or "none"). Empty → the default set (process, os, host).
	ResourceDetectors string

	// extraAttrs are additional resource attributes merged over the detected
	// resource (set via WithResourceAttrs).
	extraAttrs []attribute.KeyValue

	// errorHandler overrides the default stderr export-error handler.
	errorHandler func(error)

	// envOverrides, when false, makes programmatic/preset values win over raw
	// OTEL_* environment variables. Default true (spec precedence).
	envOverrides bool
	// selfTest, set by WithSelfTest, makes Init send one span synchronously and
	// fail loudly if it can't be exported.
	selfTest bool
	// dryRun, set by WithDryRun, prints the resolved config and exports to
	// stdout instead of the configured backend.
	dryRun bool
}

// SignalConfig configures one signal's exporter. A disabled signal yields a
// no-op provider.
type SignalConfig struct {
	// Enabled turns the signal on. A zero Config has all signals disabled;
	// presets and With* options enable them.
	Enabled bool
	// Transport selects stdout / HTTP / gRPC.
	Transport Transport
	// Endpoint is a host, host:port, or URL. otelkit derives the correct port
	// and OTLP path from it (see resolveEndpoint).
	Endpoint string
	// EndpointIsURL marks Endpoint as a complete, per-signal URL to be used
	// verbatim — otelkit will NOT append /v1/<signal> to it. Mirrors the OTLP
	// per-signal-endpoint rule.
	EndpointIsURL bool
	// Headers carries auth and routing headers (e.g. authorization, dataset).
	Headers map[string]string
	// BatchTimeout overrides the batch/schedule delay for this signal. 0 →
	// the spec default.
	BatchTimeout time.Duration
}

// defaultConfig returns a Config with spec-compliant defaults and all signals
// disabled (presets/options enable them).
func defaultConfig() Config {
	return Config{
		Sampler:           SamplerParentBasedAlwaysOn,
		SamplerRatio:      1.0,
		MetricTemporality: TemporalityCumulative,
		MetricInterval:    defaultMetricInterval,
		TLS:               TLSModeTLS,
		envOverrides:      true,
	}
}

// resolveEndpoint computes the final endpoint string an exporter should use
// for sig, owning all port and OTLP-path construction so callers never have to
// reason about 4317-vs-4318 or whether /v1/<signal> is appended.
//
// Rules (per the OTLP exporter spec):
//   - gRPC: host:port only, never a path; default port 4317.
//   - HTTP, EndpointIsURL=true: the URL is used verbatim (no /v1 append).
//   - HTTP, generic endpoint: ensure scheme + port, then append /v1/<signal>
//     to the existing base path, guarding against double-append.
//   - stdout: no endpoint.
func (sc SignalConfig) resolveEndpoint(sig Signal, tls TLSMode) (string, error) {
	ep := strings.TrimSpace(sc.Endpoint)
	switch sc.Transport {
	case TransportStdout:
		return "", nil
	case TransportGRPC:
		return grpcHostPort(ep)
	case TransportHTTP:
		return httpURL(ep, sig, sc.EndpointIsURL, tls)
	default:
		return "", fmt.Errorf("%w: %s", ErrInvalidProtocol, sc.Transport)
	}
}

// GRPCTarget returns the bare host:port an OTLP/gRPC exporter should dial for
// this signal, applying the default OTLP gRPC port (4317) when none is present.
// It is exported for the contrib/otelkit-grpc module's exporter factories.
func (sc SignalConfig) GRPCTarget() (string, error) {
	return grpcHostPort(sc.Endpoint)
}

// grpcHostPort normalizes ep to a bare host:port for the gRPC exporter,
// stripping any scheme or path and defaulting the port to 4317.
func grpcHostPort(ep string) (string, error) {
	if ep == "" {
		return "", ErrMissingEndpoint
	}
	if strings.Contains(ep, "://") {
		u, err := url.Parse(ep)
		if err != nil {
			return "", fmt.Errorf("otelkit: parse grpc endpoint %q: %w", ep, err)
		}
		ep = u.Host
		if ep == "" {
			ep = u.Path // tolerate "grpc:host:port"-ish inputs
		}
	}
	if !strings.Contains(ep, ":") {
		ep = fmt.Sprintf("%s:%d", ep, TransportGRPC.defaultPort())
	}
	return ep, nil
}

// httpURL builds the full OTLP/HTTP URL for sig, applying the path-append rule.
func httpURL(ep string, sig Signal, isURL bool, tls TLSMode) (string, error) {
	if ep == "" {
		return "", ErrMissingEndpoint
	}
	// Ensure a scheme so url.Parse populates Host rather than Path.
	if !strings.Contains(ep, "://") {
		scheme := "https"
		if tls == TLSModePlaintext {
			scheme = "http"
		}
		ep = scheme + "://" + ep
	}
	u, err := url.Parse(ep)
	if err != nil {
		return "", fmt.Errorf("otelkit: parse http endpoint %q: %w", ep, err)
	}
	if u.Host == "" {
		return "", fmt.Errorf("otelkit: http endpoint %q has no host", ep)
	}
	if isURL {
		return u.String(), nil // per-signal URL: verbatim, no port/path changes
	}
	// Default the port when the host has none.
	if u.Port() == "" {
		u.Host = fmt.Sprintf("%s:%d", u.Host, TransportHTTP.defaultPort())
	}
	suffix := sig.pathSuffix()
	base := strings.TrimRight(u.Path, "/")
	if !strings.HasSuffix(base, suffix) { // guard against double-append
		u.Path = base + suffix
	}
	return u.String(), nil
}
