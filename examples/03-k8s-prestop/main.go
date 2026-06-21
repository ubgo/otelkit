// Example 03 — Kubernetes graceful shutdown.
//
//	go run ./03-k8s-prestop
//
// RunOnSignal blocks until SIGTERM/SIGINT (what the kubelet sends on pod
// termination), then flushes all signals on a fresh context so a cancelled app
// context can't abort the flush. Pair with a terminationGracePeriodSeconds that
// exceeds your export timeout.
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/ubgo/otelkit"
)

func main() {
	ctx := context.Background()

	tel, err := otelkit.Init(ctx,
		otelkit.WithService("k8s-service", "1.0.0"),
		otelkit.WithEnvironment("prod"),
		otelkit.WithPreset(otelkit.PresetCollector("otel-collector:4318", otelkit.TransportHTTP)),
	)
	if err != nil {
		log.Fatalf("otelkit init: %v", err)
	}
	tel.SetGlobal()

	srv := &http.Server{Addr: ":8080"}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("server: %v", err)
		}
	}()

	log.Println("running; send SIGTERM to drain and flush telemetry")
	// Blocks until SIGTERM/SIGINT, then flushes telemetry. Stop the HTTP server
	// alongside it for a full graceful shutdown (see github.com/ubgo/shutdown
	// for phased multi-resource shutdown).
	if err := tel.RunOnSignal(ctx); err != nil {
		log.Printf("telemetry shutdown errors: %v", err)
	}
	_ = srv.Shutdown(context.Background())
}
