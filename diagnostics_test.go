package otelkit

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"go.opentelemetry.io/otel"
)

func TestInstallErrorHandlerCustom(t *testing.T) {
	var got error
	installErrorHandler(Config{errorHandler: func(err error) { got = err }})
	t.Cleanup(func() { installErrorHandler(Config{}) }) // restore default
	sentinel := errors.New("export failed")
	otel.Handle(sentinel)
	if !errors.Is(got, sentinel) {
		t.Errorf("handler got %v, want %v", got, sentinel)
	}
}

func TestInstallErrorHandlerDefault(t *testing.T) {
	installErrorHandler(Config{}) // default stderr handler; must not panic
	otel.Handle(errors.New("ignored"))
}

func TestInitInstallsErrorHandler(t *testing.T) {
	var got error
	tel, err := Init(context.Background(),
		WithEnvOverrides(false),
		WithPreset(PresetStdout()),
		WithService("s", "1"),
		WithErrorHandler(func(e error) { got = e }),
	)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	t.Cleanup(func() { _ = tel.Shutdown(context.Background()); installErrorHandler(Config{}) })
	otel.Handle(errors.New("boom"))
	if got == nil {
		t.Error("Init did not install the custom error handler")
	}
}

func TestProbeStdoutNoop(t *testing.T) {
	if err := ProbeEndpoint(context.Background(), "anything", TransportStdout, TLSModeTLS); err != nil {
		t.Errorf("stdout probe: %v", err)
	}
}

func TestProbeReachablePlaintext(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	if err := ProbeEndpoint(context.Background(), ln.Addr().String(), TransportHTTP, TLSModePlaintext); err != nil {
		t.Errorf("probe reachable: %v", err)
	}
}

func TestProbeUnreachable(t *testing.T) {
	// Port 1 on localhost: connection refused.
	err := ProbeEndpoint(context.Background(), "127.0.0.1:1", TransportHTTP, TLSModePlaintext)
	if err == nil {
		t.Fatal("expected unreachable error")
	}
}

func TestProbeTLSHandshakeFails(t *testing.T) {
	// A plain TCP listener that never speaks TLS → handshake fails.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		c, e := ln.Accept()
		if e == nil {
			_ = c.Close()
		}
	}()
	if err := ProbeEndpoint(context.Background(), ln.Addr().String(), TransportHTTP, TLSModeTLS); err == nil {
		t.Error("expected TLS handshake error")
	}
}

func TestProbeEndpointBadEndpoint(t *testing.T) {
	if err := ProbeEndpoint(context.Background(), "", TransportHTTP, TLSModeTLS); !errors.Is(err, ErrMissingEndpoint) {
		t.Errorf("err = %v, want ErrMissingEndpoint", err)
	}
}

func TestProbeTLSSuccessSkipVerify(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	if err := ProbeEndpoint(context.Background(), u.Host, TransportHTTP, TLSModeSkipVerify); err != nil {
		t.Errorf("probe TLS skip-verify: %v", err)
	}
}

func TestProbeHostPort(t *testing.T) {
	tests := []struct {
		ep        string
		transport Transport
		want      string
		wantErr   bool
	}{
		{"host", TransportHTTP, "host:4318", false},
		{"host:9999", TransportGRPC, "host:9999", false},
		{"https://api.example/otlp", TransportHTTP, "api.example:4318", false},
		{"https://api.example:443/x", TransportHTTP, "api.example:443", false},
		{"", TransportHTTP, "", true},
		{"://[", TransportHTTP, "", true},
		{"https:///nohost", TransportHTTP, "", true},
	}
	for _, tt := range tests {
		got, err := probeHostPort(tt.ep, tt.transport)
		if tt.wantErr {
			if err == nil {
				t.Errorf("probeHostPort(%q) err = nil, want error", tt.ep)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Errorf("probeHostPort(%q) = (%q,%v), want %q", tt.ep, got, err, tt.want)
		}
	}
}
