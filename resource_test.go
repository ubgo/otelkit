package otelkit

import (
	"context"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func TestResolveServiceName(t *testing.T) {
	if got := resolveServiceName("checkout"); got != "checkout" {
		t.Errorf("explicit = %q", got)
	}
	if got := resolveServiceName("  trimmed  "); got != "trimmed" {
		t.Errorf("trim = %q", got)
	}
	got := resolveServiceName("")
	if !strings.HasPrefix(got, "unknown_service:") {
		t.Errorf("fallback = %q, want unknown_service:<binary>", got)
	}
}

func TestDetectorOptionsCounts(t *testing.T) {
	tests := []struct {
		tokens string
		want   int
	}{
		{"", 3},                   // default: process, os, host
		{"none", 0},               // none
		{"all", 5},                // env, host, os, process, container
		{"host,os", 2},            // subset
		{"host, os , process", 3}, // whitespace tolerant
		{"host,bogus,os", 2},      // unknown token ignored
		{"NONE", 0},               // case-insensitive
	}
	for _, tt := range tests {
		if got := len(detectorOptions(tt.tokens)); got != tt.want {
			t.Errorf("detectorOptions(%q) = %d opts, want %d", tt.tokens, got, tt.want)
		}
	}
}

func TestBuildResource(t *testing.T) {
	c := defaultConfig()
	c.ServiceName = "checkout"
	c.ServiceVersion = "1.4.2"
	c.Environment = "prod"
	c.ResourceDetectors = "none" // deterministic: no host/os attrs
	c.extraAttrs = []attribute.KeyValue{attribute.String("team", "core")}

	res, err := buildResource(context.Background(), c)
	if err != nil {
		t.Fatalf("buildResource: %v", err)
	}

	want := map[attribute.Key]string{
		semconv.ServiceNameKey:    "checkout",
		semconv.ServiceVersionKey: "1.4.2",
		attrDeploymentEnvironment: "prod",
		"team":                    "core",
	}
	got := map[attribute.Key]string{}
	for _, kv := range res.Attributes() {
		got[kv.Key] = kv.Value.AsString()
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("resource[%s] = %q, want %q", k, got[k], v)
		}
	}
}

func TestBuildResourceUnknownServiceFallback(t *testing.T) {
	c := defaultConfig()
	c.ResourceDetectors = "none"
	res, err := buildResource(context.Background(), c)
	if err != nil {
		t.Fatalf("buildResource: %v", err)
	}
	var svc string
	for _, kv := range res.Attributes() {
		if kv.Key == semconv.ServiceNameKey {
			svc = kv.Value.AsString()
		}
	}
	if !strings.HasPrefix(svc, "unknown_service:") {
		t.Errorf("service.name = %q, want unknown_service:<binary>", svc)
	}
}

func TestBuildResourceWithDefaultDetectors(t *testing.T) {
	// Exercises the detector path (process/os/host) end to end.
	c := defaultConfig()
	c.ServiceName = "svc"
	res, err := buildResource(context.Background(), c)
	if err != nil {
		t.Fatalf("buildResource: %v", err)
	}
	if res == nil || len(res.Attributes()) == 0 {
		t.Error("expected a non-empty resource from default detectors")
	}
}
