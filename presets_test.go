package otelkit

import (
	"encoding/base64"
	"testing"
)

func applyPreset(p Preset) Config {
	c := defaultConfig()
	WithPreset(p)(&c)
	return c
}

func TestWithPresetNil(t *testing.T) {
	c := defaultConfig()
	WithPreset(nil)(&c) // must not panic
	if c.Traces.Enabled {
		t.Error("nil preset enabled a signal")
	}
}

func TestPresetStdout(t *testing.T) {
	c := applyPreset(PresetStdout())
	for _, sc := range []SignalConfig{c.Traces, c.Metrics, c.Logs} {
		if !sc.Enabled || sc.Transport != TransportStdout {
			t.Errorf("signal = %+v, want stdout enabled", sc)
		}
	}
}

func TestPresetHyperDX(t *testing.T) {
	c := applyPreset(PresetHyperDX("rawkey", ""))
	if c.Traces.Endpoint != "https://in-otel.hyperdx.io" {
		t.Errorf("endpoint = %q", c.Traces.Endpoint)
	}
	if c.Traces.Headers["authorization"] != "rawkey" {
		t.Errorf("auth = %v", c.Traces.Headers)
	}
	// Custom endpoint honored.
	c2 := applyPreset(PresetHyperDX("k", "https://self.hosted:4318"))
	if c2.Logs.Endpoint != "https://self.hosted:4318" {
		t.Errorf("custom endpoint = %q", c2.Logs.Endpoint)
	}
}

func TestPresetGrafanaCloud(t *testing.T) {
	c := applyPreset(PresetGrafanaCloud("12345", "tok", "https://otlp-gateway.grafana.net/otlp"))
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("12345:tok"))
	if c.Traces.Headers["Authorization"] != want {
		t.Errorf("auth = %q, want %q", c.Traces.Headers["Authorization"], want)
	}
	if c.Metrics.Endpoint != "https://otlp-gateway.grafana.net/otlp" {
		t.Errorf("endpoint = %q", c.Metrics.Endpoint)
	}
}

func TestPresetHoneycomb(t *testing.T) {
	c := applyPreset(PresetHoneycomb("hkey", "myds", ""))
	if c.Traces.Endpoint != "https://api.honeycomb.io" {
		t.Errorf("endpoint = %q", c.Traces.Endpoint)
	}
	if c.Traces.Headers["x-honeycomb-team"] != "hkey" {
		t.Errorf("traces team = %v", c.Traces.Headers)
	}
	if _, ok := c.Traces.Headers["x-honeycomb-dataset"]; ok {
		t.Error("traces should not carry dataset header")
	}
	if c.Metrics.Headers["x-honeycomb-dataset"] != "myds" || c.Metrics.Headers["x-honeycomb-team"] != "hkey" {
		t.Errorf("metrics headers = %v", c.Metrics.Headers)
	}
}

func TestPresetDatadog(t *testing.T) {
	c := applyPreset(PresetDatadog("ddkey", ""))
	if c.Traces.Endpoint != "https://otlp.datadoghq.com" {
		t.Errorf("endpoint = %q", c.Traces.Endpoint)
	}
	if c.Metrics.Headers["dd-api-key"] != "ddkey" {
		t.Errorf("auth = %v", c.Metrics.Headers)
	}
	if c.MetricTemporality != TemporalityDelta {
		t.Errorf("temporality = %v, want delta (Datadog rejects cumulative)", c.MetricTemporality)
	}
}

func TestPresetNewRelic(t *testing.T) {
	c := applyPreset(PresetNewRelic("nrkey", ""))
	if c.Traces.Endpoint != "https://otlp.nr-data.net" {
		t.Errorf("endpoint = %q", c.Traces.Endpoint)
	}
	if c.Traces.Headers["api-key"] != "nrkey" {
		t.Errorf("auth = %v", c.Traces.Headers)
	}
	if c.MetricTemporality != TemporalityDelta {
		t.Errorf("temporality = %v, want delta", c.MetricTemporality)
	}
	// Custom endpoint.
	c2 := applyPreset(PresetNewRelic("k", "https://otlp.eu01.nr-data.net"))
	if c2.Traces.Endpoint != "https://otlp.eu01.nr-data.net" {
		t.Errorf("custom = %q", c2.Traces.Endpoint)
	}
	// Honeycomb custom endpoint + Datadog custom endpoint for branch coverage.
	if got := applyPreset(PresetHoneycomb("k", "d", "https://api.eu1.honeycomb.io")).Traces.Endpoint; got != "https://api.eu1.honeycomb.io" {
		t.Errorf("honeycomb custom = %q", got)
	}
	if got := applyPreset(PresetDatadog("k", "https://agent:4318")).Traces.Endpoint; got != "https://agent:4318" {
		t.Errorf("datadog custom = %q", got)
	}
}

func TestPresetCollector(t *testing.T) {
	c := applyPreset(PresetCollector("collector:4317", TransportGRPC))
	for _, sc := range []SignalConfig{c.Traces, c.Metrics, c.Logs} {
		if !sc.Enabled || sc.Transport != TransportGRPC || sc.Endpoint != "collector:4317" {
			t.Errorf("signal = %+v", sc)
		}
		if len(sc.Headers) != 0 {
			t.Errorf("collector should have no auth headers, got %v", sc.Headers)
		}
	}
}

func TestPresetViaInit(t *testing.T) {
	// End-to-end: a preset applied through Init builds working providers.
	tel, err := Init(t.Context(), WithEnvOverrides(false), WithPreset(PresetStdout()), WithService("s", "1"))
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	_ = tel.Shutdown(t.Context())
}
