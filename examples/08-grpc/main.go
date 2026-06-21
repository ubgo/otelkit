// Example 08 — OTLP/gRPC via the contrib module.
//
//	go run ./08-grpc
//
// The core otelkit module ships HTTP + stdout only. A blank import of
// contrib/otelkit-grpc registers the gRPC exporter factories, so TransportGRPC
// (and gRPC-preferring presets) just work. Without it, TransportGRPC returns
// otelkit.ErrGRPCNotLinked — loud, not silent.
package main

import (
	"context"
	"log"

	"github.com/ubgo/otelkit"
	_ "github.com/ubgo/otelkit/contrib/otelkit-grpc" // registers OTLP/gRPC exporters
)

func main() {
	ctx := context.Background()

	tel, err := otelkit.Init(ctx,
		otelkit.WithService("grpc-example", "1.0.0"),
		otelkit.WithPreset(otelkit.PresetCollector("localhost:4317", otelkit.TransportGRPC)),
		otelkit.WithTLS(otelkit.TLSModePlaintext),
	)
	if err != nil {
		log.Fatalf("otelkit init: %v", err)
	}
	tel.SetGlobal()
	defer tel.Shutdown(ctx)

	_, span := tel.Tracer("grpc-example").Start(ctx, "hello")
	span.End()
}
