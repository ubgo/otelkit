// Example 04 — vendor presets: switch backends in one line.
//
//	OTELKIT_BACKEND=grafana go run ./04-presets
//
// Each preset encodes that vendor's endpoint, auth header name+format, path
// quirk, and metric temporality. Swapping the preset is the only change needed
// to point at a different backend.
package main

import (
	"context"
	"log"
	"os"

	"github.com/ubgo/otelkit"
)

func main() {
	ctx := context.Background()

	var preset otelkit.Preset
	switch os.Getenv("OTELKIT_BACKEND") {
	case "hyperdx":
		preset = otelkit.PresetHyperDX(os.Getenv("HYPERDX_API_KEY"), "")
	case "grafana":
		preset = otelkit.PresetGrafanaCloud(
			os.Getenv("GRAFANA_INSTANCE_ID"),
			os.Getenv("GRAFANA_TOKEN"),
			os.Getenv("GRAFANA_OTLP_ENDPOINT"), // e.g. https://otlp-gateway-<zone>.grafana.net/otlp
		)
	case "honeycomb":
		preset = otelkit.PresetHoneycomb(os.Getenv("HONEYCOMB_API_KEY"), "my-metrics-dataset", "")
	case "datadog":
		preset = otelkit.PresetDatadog(os.Getenv("DD_API_KEY"), "") // forces delta temporality
	case "newrelic":
		preset = otelkit.PresetNewRelic(os.Getenv("NEW_RELIC_LICENSE_KEY"), "")
	default:
		preset = otelkit.PresetStdout()
	}

	tel, err := otelkit.Init(ctx,
		otelkit.WithService("preset-example", "1.0.0"),
		otelkit.WithPreset(preset),
	)
	if err != nil {
		log.Fatalf("otelkit init: %v", err)
	}
	tel.SetGlobal()
	defer tel.Shutdown(ctx)

	_, span := tel.Tracer("preset-example").Start(ctx, "hello")
	span.End()
}
