package otelkit

import (
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
)

func TestOptionsApply(t *testing.T) {
	c := defaultConfig()
	c.apply(
		WithService("svc", "2.0"),
		WithEnvironment("prod"),
		WithSampler(SamplerAlwaysOff),
		WithSamplerRatio(0.5),
		WithMetricTemporality(TemporalityDelta),
		WithMetricInterval(7*time.Second),
		WithResourceDetectors("host,os"),
		WithResourceAttrs(attribute.String("team", "core"), attribute.Int("shard", 3)),
		WithTLS(TLSModeSkipVerify),
		WithProtocol(TransportHTTP),
		WithSelfTest(),
		WithDryRun(),
		WithEnvOverrides(false),
	)

	if c.ServiceName != "svc" || c.ServiceVersion != "2.0" {
		t.Errorf("service = %q/%q", c.ServiceName, c.ServiceVersion)
	}
	if c.Environment != "prod" {
		t.Errorf("env = %q", c.Environment)
	}
	if c.Sampler != SamplerAlwaysOff || c.SamplerRatio != 0.5 {
		t.Errorf("sampler = %v ratio %v", c.Sampler, c.SamplerRatio)
	}
	if c.MetricTemporality != TemporalityDelta || c.MetricInterval != 7*time.Second {
		t.Errorf("metrics = %v / %v", c.MetricTemporality, c.MetricInterval)
	}
	if c.ResourceDetectors != "host,os" {
		t.Errorf("detectors = %q", c.ResourceDetectors)
	}
	if len(c.extraAttrs) != 2 {
		t.Errorf("extraAttrs = %v", c.extraAttrs)
	}
	if c.TLS != TLSModeSkipVerify {
		t.Errorf("tls = %v", c.TLS)
	}
	if c.Traces.Transport != TransportHTTP || c.Metrics.Transport != TransportHTTP || c.Logs.Transport != TransportHTTP {
		t.Error("WithProtocol did not set all signals")
	}
	if !c.selfTest || !c.dryRun {
		t.Error("selfTest/dryRun not set")
	}
	if c.envOverrides {
		t.Error("WithEnvOverrides(false) did not take")
	}
}

func TestWithConfig(t *testing.T) {
	c := defaultConfig()
	c.apply(WithConfig(Config{
		ServiceName: "fromstruct",
		Traces:      SignalConfig{Enabled: true, Transport: TransportStdout},
	}))
	if c.ServiceName != "fromstruct" {
		t.Errorf("ServiceName = %q", c.ServiceName)
	}
	if !c.Traces.Enabled {
		t.Error("traces not enabled from struct")
	}
	// Internal defaults are restored even though the struct left them zero.
	if c.SamplerRatio != 1.0 {
		t.Errorf("SamplerRatio = %v, want restored 1.0", c.SamplerRatio)
	}
	if c.MetricInterval != defaultMetricInterval {
		t.Errorf("MetricInterval = %v, want restored default", c.MetricInterval)
	}
}

func TestWithConfigKeepsExplicitValues(t *testing.T) {
	c := defaultConfig()
	c.apply(WithConfig(Config{SamplerRatio: 0.1, MetricInterval: 2 * time.Second}))
	if c.SamplerRatio != 0.1 {
		t.Errorf("SamplerRatio = %v, want 0.1", c.SamplerRatio)
	}
	if c.MetricInterval != 2*time.Second {
		t.Errorf("MetricInterval = %v, want 2s", c.MetricInterval)
	}
}
