package otelkit

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

// probeTimeout bounds a single connectivity probe dial.
const probeTimeout = 5 * time.Second

// ProbeEndpoint checks that the OTLP endpoint is reachable, turning the SDK's
// opaque "context deadline exceeded" / gRPC "Unavailable" failures into a
// specific, human-readable diagnosis at startup. It performs a TCP dial (and,
// for TLS modes, a TLS handshake) against the endpoint's host:port.
//
// endpoint accepts the same forms as SignalConfig.Endpoint (host, host:port,
// or URL). mode selects whether a TLS handshake is attempted. A nil return
// means the first hop is reachable; it does not guarantee the backend will
// accept or store the data.
func ProbeEndpoint(ctx context.Context, endpoint string, transport Transport, mode TLSMode) error {
	if transport == TransportStdout {
		return nil
	}
	hostPort, err := probeHostPort(endpoint, transport)
	if err != nil {
		return err
	}

	dctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	var d net.Dialer
	conn, err := d.DialContext(dctx, "tcp", hostPort)
	if err != nil {
		return fmt.Errorf("otelkit: cannot reach %s — check host/port/DNS and that the collector is running: %w", hostPort, err)
	}
	defer conn.Close() //nolint:errcheck

	if mode == TLSModePlaintext {
		return nil
	}
	host, _, _ := net.SplitHostPort(hostPort)
	tconn := tls.Client(conn, &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: mode == TLSModeSkipVerify, //nolint:gosec // honors TLSModeSkipVerify
	})
	defer tconn.Close() //nolint:errcheck
	if err := tconn.HandshakeContext(dctx); err != nil {
		return fmt.Errorf("otelkit: TLS handshake to %s failed — check the certificate or use plaintext/skip-verify: %w", hostPort, err)
	}
	return nil
}

// probeHostPort extracts a dialable host:port from endpoint, applying the
// transport's default OTLP port when none is present.
func probeHostPort(endpoint string, transport Transport) (string, error) {
	ep := strings.TrimSpace(endpoint)
	if ep == "" {
		return "", ErrMissingEndpoint
	}
	if strings.Contains(ep, "://") {
		u, err := url.Parse(ep)
		if err != nil {
			return "", fmt.Errorf("otelkit: parse probe endpoint %q: %w", ep, err)
		}
		ep = u.Host
	}
	if ep == "" {
		return "", fmt.Errorf("otelkit: probe endpoint %q has no host", endpoint)
	}
	if _, _, err := net.SplitHostPort(ep); err != nil {
		ep = net.JoinHostPort(ep, fmt.Sprintf("%d", transport.defaultPort()))
	}
	return ep, nil
}
