package otelkit

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"go.opentelemetry.io/contrib/otelconf"
	"go.opentelemetry.io/otel/propagation"
)

func TestConfigFilePath(t *testing.T) {
	t.Setenv("OTEL_CONFIG_FILE", "")
	t.Setenv("OTEL_EXPERIMENTAL_CONFIG_FILE", "")
	if got := configFilePath(); got != "" {
		t.Errorf("empty = %q", got)
	}
	t.Setenv("OTEL_EXPERIMENTAL_CONFIG_FILE", "/exp.yaml")
	if got := configFilePath(); got != "/exp.yaml" {
		t.Errorf("experimental = %q", got)
	}
	t.Setenv("OTEL_CONFIG_FILE", "/std.yaml")
	if got := configFilePath(); got != "/std.yaml" {
		t.Errorf("std should win = %q", got)
	}
}

func writeFile(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestInitFromFileReadError(t *testing.T) {
	_, err := initFromFile(context.Background(), filepath.Join(t.TempDir(), "missing.yaml"))
	if err == nil || !contains(err.Error(), "read config file") {
		t.Fatalf("err = %v, want read error", err)
	}
}

func TestInitFromFileParseError(t *testing.T) {
	path := writeFile(t, "file_format: \"0.3\"\n: : bad yaml : :\n")
	_, err := initFromFile(context.Background(), path)
	if err == nil || !contains(err.Error(), "parse config file") {
		t.Fatalf("err = %v, want parse error", err)
	}
}

func TestInitFromFileNewSDKError(t *testing.T) {
	orig := newSDKFn
	t.Cleanup(func() { newSDKFn = orig })
	sentinel := errors.New("sdk boom")
	newSDKFn = func(...otelconf.ConfigurationOption) (otelconf.SDK, error) {
		return otelconf.SDK{}, sentinel
	}
	path := writeFile(t, "file_format: \"0.3\"\n")
	_, err := initFromFile(context.Background(), path)
	if !errors.Is(err, sentinel) || !contains(err.Error(), "build SDK") {
		t.Fatalf("err = %v, want wrapped sdk error", err)
	}
}

func TestInitFromFileSuccess(t *testing.T) {
	path := writeFile(t, "file_format: \"0.3\"\n")
	tel, err := initFromFile(context.Background(), path)
	if err != nil {
		t.Fatalf("initFromFile: %v", err)
	}
	if tel.TracerProvider() == nil || tel.LoggerProvider() == nil || tel.MeterProvider() == nil {
		t.Error("nil provider from declarative config")
	}
	if tel.propagator == nil {
		t.Error("nil propagator")
	}
	_ = tel.Shutdown(context.Background())
}

func TestPropagatorOrDefault(t *testing.T) {
	if propagatorOrDefault(nil) == nil {
		t.Error("nil input should yield default propagator")
	}
	tc := propagation.TraceContext{}
	if got := propagatorOrDefault(tc); got != tc {
		t.Error("non-nil propagator should pass through unchanged")
	}
}

func TestInitDelegatesToFile(t *testing.T) {
	path := writeFile(t, "file_format: \"0.3\"\n")
	t.Setenv("OTEL_CONFIG_FILE", path)
	tel, err := Init(context.Background(), WithService("ignored", "1"))
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if tel.TracerProvider() == nil {
		t.Error("nil tracer provider from delegated init")
	}
	_ = tel.Shutdown(context.Background())
}
