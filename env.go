package otelkit

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// envDisabled reports whether OTEL_SDK_DISABLED is set to a truthy value
// (case-insensitive "true"). Per spec, any other value (including unset) means
// the SDK is enabled.
func envDisabled() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("OTEL_SDK_DISABLED")), "true")
}

// applyEnv overlays the standard OTEL_* environment variables onto c. It is
// called only when c.envOverrides is true (the default), giving env the spec
// precedence over programmatic/preset values. Returns an error for an invalid
// OTEL_EXPORTER_OTLP_PROTOCOL value.
func applyEnv(c *Config) error {
	if v := getenv("OTEL_SERVICE_NAME"); v != "" {
		c.ServiceName = v
	}
	if v := getenv("OTEL_TRACES_SAMPLER"); v != "" {
		if s, ok := parseSampler(v); ok {
			c.Sampler = s
		}
	}
	if v := getenv("OTEL_TRACES_SAMPLER_ARG"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			c.SamplerRatio = f
		}
	}
	if v := getenv("OTEL_METRIC_EXPORT_INTERVAL"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms > 0 {
			c.MetricInterval = time.Duration(ms) * time.Millisecond
		}
	}
	if v := getenv("OTEL_EXPORTER_OTLP_METRICS_TEMPORALITY_PREFERENCE"); v != "" {
		if strings.EqualFold(v, "delta") {
			c.MetricTemporality = TemporalityDelta
		} else {
			c.MetricTemporality = TemporalityCumulative
		}
	}
	if v := getenv("OTEL_EXPORTER_OTLP_INSECURE"); strings.EqualFold(v, "true") {
		c.TLS = TLSModePlaintext
	}

	// Generic exporter settings, overridable per signal.
	genericProto := getenv("OTEL_EXPORTER_OTLP_PROTOCOL")
	genericEndpoint := getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	genericHeaders := getenv("OTEL_EXPORTER_OTLP_HEADERS")

	if err := applySignalEnv(&c.Traces, "TRACES", genericProto, genericEndpoint, genericHeaders); err != nil {
		return err
	}
	if err := applySignalEnv(&c.Metrics, "METRICS", genericProto, genericEndpoint, genericHeaders); err != nil {
		return err
	}
	if err := applySignalEnv(&c.Logs, "LOGS", genericProto, genericEndpoint, genericHeaders); err != nil {
		return err
	}
	return nil
}

// applySignalEnv overlays per-signal OTLP env (falling back to the generic
// values) onto sc. A per-signal endpoint is treated as a verbatim URL per the
// OTLP spec. When any exporter env is present for a signal, it is enabled.
func applySignalEnv(sc *SignalConfig, key, genProto, genEndpoint, genHeaders string) error {
	proto := genProto
	if v := getenv("OTEL_EXPORTER_OTLP_" + key + "_PROTOCOL"); v != "" {
		proto = v
	}
	if proto != "" {
		tr, ok := parseProtocol(proto)
		if !ok {
			return fmt.Errorf("%w: %q", ErrInvalidProtocol, proto)
		}
		sc.Transport = tr
		sc.Enabled = true
	}

	endpoint := genEndpoint
	perSignal := false
	if v := getenv("OTEL_EXPORTER_OTLP_" + key + "_ENDPOINT"); v != "" {
		endpoint = v
		perSignal = true
	}
	if endpoint != "" {
		sc.Endpoint = endpoint
		sc.EndpointIsURL = perSignal // per-signal endpoints are used verbatim
		sc.Enabled = true
	}

	headers := genHeaders
	if v := getenv("OTEL_EXPORTER_OTLP_" + key + "_HEADERS"); v != "" {
		headers = v
	}
	if headers != "" {
		sc.Headers = mergeHeaders(sc.Headers, parseHeaders(headers))
	}
	return nil
}

// getenv trims surrounding whitespace from an environment variable value.
func getenv(key string) string { return strings.TrimSpace(os.Getenv(key)) }

// parseProtocol maps an OTEL_EXPORTER_OTLP_PROTOCOL value to a Transport.
// Both http/protobuf and http/json map to HTTP (core uses protobuf).
func parseProtocol(v string) (Transport, bool) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "grpc":
		return TransportGRPC, true
	case "http/protobuf", "http/json":
		return TransportHTTP, true
	default:
		return 0, false
	}
}

// parseSampler maps an OTEL_TRACES_SAMPLER value to a Sampler. Unknown values
// return ok=false so the caller keeps its default.
func parseSampler(v string) (Sampler, bool) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "always_on":
		return SamplerAlwaysOn, true
	case "always_off":
		return SamplerAlwaysOff, true
	case "traceidratio":
		return SamplerTraceIDRatio, true
	case "parentbased_always_on":
		return SamplerParentBasedAlwaysOn, true
	case "parentbased_traceidratio":
		return SamplerParentBasedTraceIDRatio, true
	default:
		return 0, false
	}
}

// parseHeaders parses a W3C Baggage-style "k1=v1,k2=v2" string into a map,
// trimming whitespace around keys and values and skipping malformed pairs.
func parseHeaders(s string) map[string]string {
	out := map[string]string{}
	for _, pair := range strings.Split(s, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		k, v, ok := strings.Cut(pair, "=")
		k = strings.TrimSpace(k)
		if !ok || k == "" {
			continue
		}
		out[k] = strings.TrimSpace(v)
	}
	return out
}

// mergeHeaders returns a new map with add overlaid on base (add wins).
func mergeHeaders(base, add map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(add))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range add {
		out[k] = v
	}
	return out
}
