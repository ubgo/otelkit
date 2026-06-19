package otelkit

import (
	"time"

	"go.opentelemetry.io/otel/attribute"
)

// Option configures a Config before providers are built. Options are applied
// after any Preset and before the OTEL_* environment overlay (unless
// WithEnvOverrides(false) is set), giving the precedence: preset < options <
// env.
type Option func(*Config)

// WithConfig replaces the working Config wholesale. Use it when you already
// have a fully-formed Config (e.g. mapped from PKL); later options still apply
// on top.
func WithConfig(c Config) Option {
	return func(dst *Config) {
		envOverrides := dst.envOverrides
		*dst = c
		// Preserve the resolver's internal defaults unless c set them.
		if dst.SamplerRatio == 0 {
			dst.SamplerRatio = 1.0
		}
		if dst.MetricInterval == 0 {
			dst.MetricInterval = defaultMetricInterval
		}
		dst.envOverrides = envOverrides
	}
}

// WithService sets service.name and service.version.
func WithService(name, version string) Option {
	return func(c *Config) {
		c.ServiceName = name
		c.ServiceVersion = version
	}
}

// WithEnvironment sets deployment.environment.name (e.g. "prod", "staging").
func WithEnvironment(env string) Option {
	return func(c *Config) { c.Environment = env }
}

// WithSampler sets the head sampler. For ratio samplers, pair with WithSamplerRatio.
func WithSampler(s Sampler) Option {
	return func(c *Config) { c.Sampler = s }
}

// WithSamplerRatio sets the [0,1] probability used by the ratio samplers.
func WithSamplerRatio(ratio float64) Option {
	return func(c *Config) { c.SamplerRatio = ratio }
}

// WithMetricTemporality selects cumulative (default) or delta aggregation.
func WithMetricTemporality(t Temporality) Option {
	return func(c *Config) { c.MetricTemporality = t }
}

// WithMetricInterval sets the periodic metric reader interval.
func WithMetricInterval(d time.Duration) Option {
	return func(c *Config) { c.MetricInterval = d }
}

// WithResourceDetectors sets the detector token list ("env,host,os,process,
// container", "all", or "none").
func WithResourceDetectors(tokens string) Option {
	return func(c *Config) { c.ResourceDetectors = tokens }
}

// WithResourceAttrs appends extra resource attributes, merged over the
// detected resource.
func WithResourceAttrs(attrs ...attribute.KeyValue) Option {
	return func(c *Config) { c.extraAttrs = append(c.extraAttrs, attrs...) }
}

// WithTLS sets the transport security mode for network exporters.
func WithTLS(mode TLSMode) Option {
	return func(c *Config) { c.TLS = mode }
}

// WithErrorHandler overrides the default stderr handler for telemetry export
// errors (the loud-by-default diagnostic). Pass your logger here to route
// export failures into your logging pipeline.
func WithErrorHandler(fn func(error)) Option {
	return func(c *Config) { c.errorHandler = fn }
}

// WithEnvOverrides controls whether OTEL_* environment variables override
// programmatic/preset values. Default true (spec precedence). Set false to
// make options/PKL authoritative.
func WithEnvOverrides(enabled bool) Option {
	return func(c *Config) { c.envOverrides = enabled }
}

// WithSelfTest enables the opt-in boot self-test: Init sends one span
// synchronously and surfaces any export error, so a misconfigured backend
// fails loudly at startup instead of silently dropping telemetry.
func WithSelfTest() Option {
	return func(c *Config) { c.selfTest = true }
}

// WithDryRun prints the resolved effective configuration (auth headers
// redacted) and routes telemetry to stdout instead of exporting. Use it to
// verify wiring without a backend.
func WithDryRun() Option {
	return func(c *Config) { c.dryRun = true }
}

// WithProtocol sets the transport for every enabled signal in one call,
// deriving the correct port and path. A convenience over setting each
// SignalConfig.Transport.
func WithProtocol(t Transport) Option {
	return func(c *Config) {
		c.Traces.Transport = t
		c.Metrics.Transport = t
		c.Logs.Transport = t
	}
}

// apply runs opts over c in order.
func (c *Config) apply(opts ...Option) {
	for _, o := range opts {
		o(c)
	}
}
