// errors.go — the exported sentinel errors callers can match with errors.Is.
//
// Kept in one place so the error surface is discoverable. ErrGRPCNotLinked is
// the deliberate "loud, not silent" signal returned when TransportGRPC is
// selected without importing contrib/otelkit-grpc.

package otelkit

import "errors"

// Sentinel errors returned by Init and config validation. Callers can match
// them with errors.Is.
var (
	// ErrMissingServiceName is returned when no service name is set via
	// WithService, the Config, or OTEL_SERVICE_NAME. otelkit still falls back
	// to "unknown_service:<binary>" for the resource per spec, but an
	// explicitly-required service name (validation mode) surfaces this.
	ErrMissingServiceName = errors.New("otelkit: service name is required")

	// ErrMissingEndpoint is returned when a network transport (HTTP/gRPC) is
	// selected for an enabled signal but no endpoint was provided.
	ErrMissingEndpoint = errors.New("otelkit: endpoint is required for http/grpc transport")

	// ErrInvalidProtocol is returned for an OTEL_EXPORTER_OTLP_PROTOCOL value
	// that is not one of grpc, http/protobuf, http/json.
	ErrInvalidProtocol = errors.New("otelkit: invalid OTLP protocol")

	// ErrContradictoryTLS is returned when the TLS mode contradicts the
	// supplied TLS material (e.g. plaintext with a client certificate).
	ErrContradictoryTLS = errors.New("otelkit: contradictory TLS configuration")

	// ErrGRPCNotLinked is returned when TransportGRPC is requested but no gRPC
	// exporter factory has been registered. Import the contrib/otelkit-grpc
	// module (which registers itself) to enable gRPC.
	ErrGRPCNotLinked = errors.New("otelkit: gRPC transport requires importing github.com/ubgo/otelkit/contrib/otelkit-grpc")
)
