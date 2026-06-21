// Example 01 — basic: traces to a local OTLP/HTTP collector.
//
//	go run ./01-basic
//
// Sends one span to http://localhost:4318. With no collector running the
// export fails silently — see 05-self-test for catching that at startup.
package main

import (
	"context"
	"log"

	"github.com/ubgo/otelkit"
)

func main() {
	ctx := context.Background()

	tel, err := otelkit.Init(ctx,
		otelkit.WithService("basic-example", "1.0.0"),
		otelkit.WithPreset(otelkit.PresetCollector("localhost:4318", otelkit.TransportHTTP)),
		otelkit.WithTLS(otelkit.TLSModePlaintext),
	)
	if err != nil {
		log.Fatalf("otelkit init: %v", err)
	}
	tel.SetGlobal()
	defer tel.Shutdown(ctx)

	_, span := tel.Tracer("basic-example").Start(ctx, "hello")
	span.End()

	log.Println("emitted one span; flushing on shutdown")
}
