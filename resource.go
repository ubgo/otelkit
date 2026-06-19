package otelkit

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// attrDeploymentEnvironment is the current semantic-convention key for the
// deployment environment. The older "deployment.environment" was renamed to
// "deployment.environment.name"; we emit only the new form.
const attrDeploymentEnvironment = "deployment.environment.name"

// resolveServiceName returns the configured service name, or the spec-mandated
// "unknown_service:<binary>" fallback when none is set, so a backend never
// receives a blank service.name.
func resolveServiceName(name string) string {
	name = strings.TrimSpace(name)
	if name != "" {
		return name
	}
	bin := "go"
	if len(os.Args) > 0 && os.Args[0] != "" {
		bin = filepath.Base(os.Args[0])
	}
	return "unknown_service:" + bin
}

// detectorOptions maps a detector token list to resource.New options.
//
//	""                              -> default set: process, os, host
//	"none"                          -> no detectors
//	"all"                           -> env, host, os, process, container
//	"env,host,os,process,container" -> the named subset
//
// Unknown tokens are ignored.
func detectorOptions(tokens string) []resource.Option {
	tokens = strings.TrimSpace(tokens)
	var set []string
	switch strings.ToLower(tokens) {
	case "":
		set = []string{"process", "os", "host"}
	case "none":
		set = nil
	case "all":
		set = []string{"env", "host", "os", "process", "container"}
	default:
		for _, tok := range strings.Split(tokens, ",") {
			set = append(set, strings.ToLower(strings.TrimSpace(tok)))
		}
	}

	var opts []resource.Option
	for _, tok := range set {
		switch tok {
		case "env":
			opts = append(opts, resource.WithFromEnv())
		case "host":
			opts = append(opts, resource.WithHost())
		case "os":
			opts = append(opts, resource.WithOS())
		case "process":
			opts = append(opts, resource.WithProcess())
		case "container":
			opts = append(opts, resource.WithContainer())
		}
	}
	return opts
}

// buildResource assembles the OTEL resource from c: the detected attributes
// (host/os/process/…), then the explicit service identity and any extra
// attributes merged on top so the configured service.name always wins.
func buildResource(ctx context.Context, c Config) (*resource.Resource, error) {
	identity := []attribute.KeyValue{
		semconv.ServiceName(resolveServiceName(c.ServiceName)),
	}
	if v := strings.TrimSpace(c.ServiceVersion); v != "" {
		identity = append(identity, semconv.ServiceVersion(v))
	}
	if v := strings.TrimSpace(c.Environment); v != "" {
		identity = append(identity, attribute.String(attrDeploymentEnvironment, v))
	}

	opts := detectorOptions(c.ResourceDetectors)
	// Explicit identity + extra attrs are appended last so they take
	// precedence over anything a detector produced.
	opts = append(opts, resource.WithAttributes(identity...))
	if len(c.extraAttrs) > 0 {
		opts = append(opts, resource.WithAttributes(c.extraAttrs...))
	}

	return resource.New(ctx, opts...)
}
