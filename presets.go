package otelkit

import "encoding/base64"

// Preset configures a Config for a specific backend, encoding that vendor's
// endpoint, transport, auth-header name/format, path quirk, and temporality as
// data — so switching backends is a one-line change. Apply with WithPreset.
type Preset func(*Config)

// WithPreset applies a vendor Preset. Presets run before other options, so an
// explicit With* still overrides a preset value (precedence: preset < options
// < env). A nil preset is a no-op.
func WithPreset(p Preset) Option {
	return func(c *Config) {
		if p != nil {
			p(c)
		}
	}
}

// httpSignal returns a SignalConfig for an HTTP backend with the given
// endpoint and headers.
func httpSignal(endpoint string, headers map[string]string) SignalConfig {
	return SignalConfig{
		Enabled:   true,
		Transport: TransportHTTP,
		Endpoint:  endpoint,
		Headers:   headers,
	}
}

// enableHTTP sets all three signals to the same HTTP endpoint + headers.
func enableHTTP(c *Config, endpoint string, headers map[string]string) {
	c.Traces = httpSignal(endpoint, headers)
	c.Metrics = httpSignal(endpoint, headers)
	c.Logs = httpSignal(endpoint, headers)
}

// PresetStdout routes all three signals to stdout — for local development.
func PresetStdout() Preset {
	return func(c *Config) {
		c.Traces = SignalConfig{Enabled: true, Transport: TransportStdout}
		c.Metrics = SignalConfig{Enabled: true, Transport: TransportStdout}
		c.Logs = SignalConfig{Enabled: true, Transport: TransportStdout}
	}
}

// PresetHyperDX configures HyperDX / ClickStack ingest. Auth is the raw
// ingestion key in the "authorization" header (no "Bearer" prefix). A blank
// endpoint defaults to the cloud gateway.
func PresetHyperDX(apiKey, endpoint string) Preset {
	if endpoint == "" {
		endpoint = "https://in-otel.hyperdx.io"
	}
	return func(c *Config) {
		enableHTTP(c, endpoint, map[string]string{"authorization": apiKey})
	}
}

// PresetGrafanaCloud configures Grafana Cloud OTLP. Auth is HTTP Basic with the
// base64 of "instanceID:token". The endpoint must be the "/otlp" base; otelkit
// appends "/v1/<signal>" automatically.
func PresetGrafanaCloud(instanceID, token, endpoint string) Preset {
	cred := base64.StdEncoding.EncodeToString([]byte(instanceID + ":" + token))
	return func(c *Config) {
		enableHTTP(c, endpoint, map[string]string{"Authorization": "Basic " + cred})
	}
}

// PresetHoneycomb configures Honeycomb ingest. Auth is the team/ingest key in
// "x-honeycomb-team"; metrics additionally require the dataset in
// "x-honeycomb-dataset". A blank endpoint defaults to the US gateway.
func PresetHoneycomb(apiKey, dataset, endpoint string) Preset {
	if endpoint == "" {
		endpoint = "https://api.honeycomb.io"
	}
	return func(c *Config) {
		base := map[string]string{"x-honeycomb-team": apiKey}
		enableHTTP(c, endpoint, base)
		// Metrics carry the dataset header in addition to the team key.
		c.Metrics.Headers = map[string]string{
			"x-honeycomb-team":    apiKey,
			"x-honeycomb-dataset": dataset,
		}
	}
}

// PresetDatadog configures Datadog OTLP intake. Auth is the API key in
// "dd-api-key". Datadog's direct intake rejects cumulative metrics, so the
// preset forces delta temporality. A blank endpoint defaults to the intake host.
func PresetDatadog(apiKey, endpoint string) Preset {
	if endpoint == "" {
		endpoint = "https://otlp.datadoghq.com"
	}
	return func(c *Config) {
		enableHTTP(c, endpoint, map[string]string{"dd-api-key": apiKey})
		c.MetricTemporality = TemporalityDelta
	}
}

// PresetNewRelic configures New Relic OTLP. Auth is the ingest license key in
// "api-key"; New Relic strongly prefers delta temporality. A blank endpoint
// defaults to the US collector.
func PresetNewRelic(licenseKey, endpoint string) Preset {
	if endpoint == "" {
		endpoint = "https://otlp.nr-data.net"
	}
	return func(c *Config) {
		enableHTTP(c, endpoint, map[string]string{"api-key": licenseKey})
		c.MetricTemporality = TemporalityDelta
	}
}

// PresetCollector configures a generic OTLP collector with no auth — the
// vendor-neutral escape hatch. Choose the transport (HTTP or gRPC); gRPC
// requires importing contrib/otelkit-grpc.
func PresetCollector(endpoint string, transport Transport) Preset {
	return func(c *Config) {
		sc := SignalConfig{Enabled: true, Transport: transport, Endpoint: endpoint}
		c.Traces = sc
		c.Metrics = sc
		c.Logs = sc
	}
}
