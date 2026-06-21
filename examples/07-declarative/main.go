// Example 07 — declarative config: drive otelkit from an OTEL config file.
//
//	OTEL_CONFIG_FILE=./otel-config.yaml go run ./07-declarative
//
// When OTEL_CONFIG_FILE (or OTEL_EXPERIMENTAL_CONFIG_FILE) is set, otelkit
// delegates to the stable opentelemetry-configuration loader (otelconf). The
// file wins outright — programmatic options and flat OTEL_* vars are ignored,
// except ${ENV} substitution inside the YAML.
package main

import (
	"context"
	"log"

	"github.com/ubgo/otelkit"
)

func main() {
	ctx := context.Background()

	// With OTEL_CONFIG_FILE set, WithService below is ignored — the file is the
	// source of truth.
	tel, err := otelkit.Init(ctx, otelkit.WithService("ignored-when-file-set", "1.0.0"))
	if err != nil {
		log.Fatalf("otelkit init: %v", err)
	}
	tel.SetGlobal()
	defer tel.Shutdown(ctx)

	_, span := tel.Tracer("declarative-example").Start(ctx, "hello")
	span.End()
}
